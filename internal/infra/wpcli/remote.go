package wpcli

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/sibukixxx/wp2emdash/internal/shell"
)

type RemoteConfig struct {
	Target string
	Port   int
	Key    string
	WPRoot string
}

func (a *Auditor) isRemote() bool {
	return a.remote != nil
}

func (a *Auditor) remotePath(elem ...string) string {
	if !a.isRemote() {
		return ""
	}
	parts := make([]string, 0, len(elem)+1)
	parts = append(parts, a.WPRoot)
	parts = append(parts, elem...)
	return path.Join(parts...)
}

func (a *Auditor) remoteOutput(ctx context.Context, code, script string) string {
	args := make([]string, 0, 8)
	args = append(args, "-o", "BatchMode=yes")
	if a.remote.Port > 0 {
		args = append(args, "-p", strconv.Itoa(a.remote.Port))
	}
	if a.remote.Key != "" {
		args = append(args, "-i", a.remote.Key)
	}
	args = append(args, "--", a.remote.Target, "sh -lc "+shell.QuotePOSIX(script))

	out, err := a.Runner.Output(ctx, "ssh", args...)
	if err != nil {
		a.warnf(code, "ssh probe failed: %v", err)
		return ""
	}
	return out
}

func (a *Auditor) remoteWP(ctx context.Context, code string, args ...string) string {
	cmd := shellCommand("wp", args...)
	script := "cd " + shell.QuotePOSIX(a.WPRoot) + " && " + cmd
	return a.remoteOutput(ctx, code, script)
}

func (a *Auditor) remoteFileExists(ctx context.Context, code, file string) bool {
	out := a.remoteOutput(ctx, code, "if [ -f "+shell.QuotePOSIX(file)+" ]; then printf yes; fi")
	return out == "yes"
}

func (a *Auditor) remoteDirExists(ctx context.Context, code, dir string) bool {
	out := a.remoteOutput(ctx, code, "if [ -d "+shell.QuotePOSIX(dir)+" ]; then printf yes; fi")
	return out == "yes"
}

func (a *Auditor) remoteDirSizeAndCount(ctx context.Context, code, dir string) (int64, int, bool) {
	script := remotePHPCommand(`
$root = $argv[1];
$size = 0;
$count = 0;
$exists = is_dir($root);
if ($exists) {
    try {
        $it = new RecursiveIteratorIterator(
            new RecursiveDirectoryIterator($root, FilesystemIterator::SKIP_DOTS)
        );
        foreach ($it as $file) {
            try {
                if (!$file->isFile()) {
                    continue;
                }
                $size += $file->getSize();
                $count++;
            } catch (Throwable $t) {
                continue;
            }
        }
    } catch (Throwable $t) {
    }
}
echo json_encode(['exists' => $exists, 'size' => $size, 'count' => $count]);
`, dir)
	out := a.remoteOutput(ctx, code, script)
	if out == "" {
		return 0, 0, false
	}
	var payload struct {
		Exists bool  `json:"exists"`
		Size   int64 `json:"size"`
		Count  int   `json:"count"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		a.warnf(code+".invalid", "remote uploads probe returned invalid JSON: %v", err)
		return 0, 0, false
	}
	return payload.Size, payload.Count, payload.Exists
}

func (a *Auditor) remoteCountFilesByExt(ctx context.Context, code, root, ext string) int {
	return a.remotePHPInt(ctx, code, `
$root = $argv[1];
$ext = ltrim(strtolower($argv[2]), '.');
$count = 0;
if (is_dir($root)) {
    try {
        $it = new RecursiveIteratorIterator(
            new RecursiveDirectoryIterator($root, FilesystemIterator::SKIP_DOTS)
        );
        foreach ($it as $file) {
            try {
                if ($file->isFile() && strtolower(pathinfo($file->getFilename(), PATHINFO_EXTENSION)) === $ext) {
                    $count++;
                }
            } catch (Throwable $t) {
                continue;
            }
        }
    } catch (Throwable $t) {
    }
}
echo $count;
`, root, ext)
}

func (a *Auditor) remoteGrepCount(ctx context.Context, code, root string, needles ...string) int {
	payload, _ := json.Marshal(needles)
	return a.remotePHPInt(ctx, code, `
$root = $argv[1];
$needles = json_decode($argv[2], true) ?: [];
$count = 0;
if (is_dir($root)) {
    try {
        $it = new RecursiveIteratorIterator(
            new RecursiveDirectoryIterator($root, FilesystemIterator::SKIP_DOTS)
        );
        foreach ($it as $file) {
            try {
                if (!$file->isFile()) {
                    continue;
                }
                $ext = strtolower(pathinfo($file->getFilename(), PATHINFO_EXTENSION));
                if ($ext !== '' && !in_array($ext, ['php','html','htm','js','jsx','ts','tsx','css','scss','json','yml','yaml','md','txt','xml','sh','env'], true)) {
                    continue;
                }
                $fh = @fopen($file->getPathname(), 'r');
                if (!$fh) {
                    continue;
                }
                while (($line = fgets($fh)) !== false) {
                    foreach ($needles as $needle) {
                        if (strpos($line, $needle) !== false) {
                            $count++;
                        }
                    }
                }
                fclose($fh);
            } catch (Throwable $t) {
                continue;
            }
        }
    } catch (Throwable $t) {
    }
}
echo $count;
`, root, string(payload))
}

func (a *Auditor) remoteGrepCountInRoots(ctx context.Context, code string, roots []string, needles ...string) int {
	total := 0
	for i, root := range roots {
		total += a.remoteGrepCount(ctx, fmt.Sprintf("%s.%d", code, i), root, needles...)
	}
	return total
}

func (a *Auditor) remoteCountLinesMatching(ctx context.Context, code, file string, needles []string) int {
	payload, _ := json.Marshal(needles)
	return a.remotePHPInt(ctx, code, `
$file = $argv[1];
$needles = array_map('strtolower', json_decode($argv[2], true) ?: []);
$count = 0;
if (is_file($file)) {
    $fh = @fopen($file, 'r');
    if ($fh) {
        while (($line = fgets($fh)) !== false) {
            $lc = strtolower($line);
            foreach ($needles as $needle) {
                if (strpos($lc, $needle) !== false) {
                    $count++;
                    break;
                }
            }
        }
        fclose($fh);
    }
}
echo $count;
`, file, string(payload))
}

func (a *Auditor) remotePHPInt(ctx context.Context, code, php string, args ...string) int {
	out := strings.TrimSpace(a.remoteOutput(ctx, code, remotePHPCommand(php, args...)))
	if out == "" {
		return 0
	}
	n, err := strconv.Atoi(out)
	if err != nil {
		a.warnf(code+".invalid", "remote PHP probe returned non-integer output: %q", out)
		return 0
	}
	return n
}

func remotePHPCommand(php string, args ...string) string {
	parts := []string{"php", "-r", php}
	if len(args) > 0 {
		parts = append(parts, "--")
		parts = append(parts, args...)
	}
	return shellCommand(parts[0], parts[1:]...)
}

func shellCommand(name string, args ...string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shell.QuotePOSIX(name))
	for _, arg := range args {
		parts = append(parts, shell.QuotePOSIX(arg))
	}
	return strings.Join(parts, " ")
}
