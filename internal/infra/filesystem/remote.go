package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"sort"
	"strings"

	"github.com/sibukixxx/wp2emdash/internal/domain/media"
	"github.com/sibukixxx/wp2emdash/internal/shell"
)

type RemoteScanConfig struct {
	Target string
	Port   int
	Key    string
	Dir    string
}

type remoteFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Ext    string `json:"ext"`
	SHA256 string `json:"sha256,omitempty"`
}

type remoteManifest struct {
	BaseDir    string         `json:"base_dir"`
	TotalFiles int            `json:"total_files"`
	TotalBytes int64          `json:"total_bytes"`
	Extensions map[string]int `json:"extensions"`
	Files      []remoteFile   `json:"files,omitempty"`
}

func ScanRemote(cfg RemoteScanConfig, opt ScanOptions) (media.Manifest, error) {
	if strings.TrimSpace(cfg.Target) == "" {
		return media.Manifest{}, fmt.Errorf("ssh target is required")
	}
	if strings.TrimSpace(cfg.Dir) == "" {
		return media.Manifest{}, fmt.Errorf("scan dir is required")
	}

	script := remoteScanPHP(opt)
	cmd := remoteSSHCommand(cfg, "php -r "+shell.QuotePOSIX(script)+" -- "+
		shell.QuotePOSIX(cfg.Dir)+" "+
		shell.QuotePOSIX(boolString(opt.Hash))+" "+
		shell.QuotePOSIX(boolString(opt.WithFiles))+" "+
		shell.QuotePOSIX(fmt.Sprintf("%d", opt.MaxFiles)))

	out, err := shell.Runner{}.Output(context.Background(), "ssh", cmd...)
	if err != nil {
		return media.Manifest{}, fmt.Errorf("remote media scan: %w", err)
	}

	var rm remoteManifest
	if err := json.Unmarshal([]byte(out), &rm); err != nil {
		return media.Manifest{}, fmt.Errorf("decode remote media manifest: %w", err)
	}

	manifest := media.Manifest{
		BaseDir:    rm.BaseDir,
		TotalFiles: rm.TotalFiles,
		TotalBytes: rm.TotalBytes,
		Extensions: rm.Extensions,
	}
	if manifest.Extensions == nil {
		manifest.Extensions = map[string]int{}
	}
	for _, rf := range rm.Files {
		manifest.Files = append(manifest.Files, media.File{
			Path:   rf.Path,
			Size:   rf.Size,
			Ext:    rf.Ext,
			MIME:   mime.TypeByExtension("." + rf.Ext),
			SHA256: rf.SHA256,
		})
	}
	sort.SliceStable(manifest.Files, func(i, j int) bool {
		return manifest.Files[i].Path < manifest.Files[j].Path
	})
	return manifest, nil
}

func remoteSSHCommand(cfg RemoteScanConfig, remoteCmd string) []string {
	args := []string{"-o", "BatchMode=yes"}
	if cfg.Port > 0 {
		args = append(args, "-p", fmt.Sprintf("%d", cfg.Port))
	}
	if cfg.Key != "" {
		args = append(args, "-i", cfg.Key)
	}
	args = append(args, cfg.Target, "sh -lc "+shell.QuotePOSIX(remoteCmd))
	return args
}

func boolString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func remoteScanPHP(opt ScanOptions) string {
	return `
$dir = $argv[1];
$withHash = ($argv[2] ?? "0") === "1";
$withFiles = ($argv[3] ?? "1") === "1";
$maxFiles = intval($argv[4] ?? "0");
$result = [
    'base_dir' => $dir,
    'total_files' => 0,
    'total_bytes' => 0,
    'extensions' => [],
    'files' => [],
];
if (is_dir($dir)) {
    try {
        $it = new RecursiveIteratorIterator(
            new RecursiveDirectoryIterator($dir, FilesystemIterator::SKIP_DOTS)
        );
        foreach ($it as $file) {
            try {
                if (!$file->isFile()) {
                    continue;
                }
                $size = $file->getSize();
                $result['total_files']++;
                $result['total_bytes'] += $size;
                $ext = strtolower(pathinfo($file->getFilename(), PATHINFO_EXTENSION));
                if (!isset($result['extensions'][$ext])) {
                    $result['extensions'][$ext] = 0;
                }
                $result['extensions'][$ext]++;
                if ($withFiles) {
                    $rel = ltrim(str_replace('\\', '/', substr($file->getPathname(), strlen(rtrim($dir, DIRECTORY_SEPARATOR)))), '/');
                    $entry = [
                        'path' => $rel,
                        'size' => $size,
                        'ext' => $ext,
                    ];
                    if ($withHash) {
                        $hash = @hash_file('sha256', $file->getPathname());
                        if ($hash !== false) {
                            $entry['sha256'] = $hash;
                        }
                    }
                    $result['files'][] = $entry;
                    if ($maxFiles > 0 && count($result['files']) >= $maxFiles) {
                        break;
                    }
                }
            } catch (Throwable $t) {
                continue;
            }
        }
    } catch (Throwable $t) {
    }
}
echo json_encode($result);
`
}
