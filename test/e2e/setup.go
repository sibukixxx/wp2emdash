package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type CLI struct {
	BinaryPath string
	RepoRoot   string
	FixtureDir string
	ToolBinDir string
}

type Result struct {
	Stdout string
	Stderr string
	Err    error
}

func NewCLI(t *testing.T) *CLI {
	t.Helper()

	repoRoot := repoRoot(t)
	fixtureDir := filepath.Join(repoRoot, "test", "e2e", "testdata", "wp-site")
	toolBinDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(toolBinDir, 0o755); err != nil {
		t.Fatalf("mkdir tool bin: %v", err)
	}

	writeStubTool(t, toolBinDir, "wp", fakeWP)
	writeStubTool(t, toolBinDir, "php", fakePHP)
	writeStubTool(t, toolBinDir, "rclone", "#!/bin/sh\nprintf '%s' \"$*\"\n")
	writeStubTool(t, toolBinDir, "wrangler", "#!/bin/sh\nexit 0\n")
	writeStubTool(t, toolBinDir, "git", "#!/bin/sh\nexit 0\n")
	writeStubTool(t, toolBinDir, "ssh", fakeSSH)

	binaryPath := filepath.Join(t.TempDir(), "wp2emdash")
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/wp2emdash")
	build.Dir = repoRoot
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-build"))
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build binary: %v\n%s", err, out)
	}

	return &CLI{
		BinaryPath: binaryPath,
		RepoRoot:   repoRoot,
		FixtureDir: fixtureDir,
		ToolBinDir: toolBinDir,
	}
}

func (c *CLI) ReplaceTool(t *testing.T, name, body string) {
	t.Helper()
	writeStubTool(t, c.ToolBinDir, name, body)
}

func (c *CLI) Run(t *testing.T, args ...string) Result {
	t.Helper()

	res := c.RunWithEnv(t, nil, args...)
	if res.Err != nil {
		t.Fatalf("run %s: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), res.Err, res.Stdout, res.Stderr)
	}
	return res
}

func (c *CLI) RunWithEnv(t *testing.T, extraEnv []string, args ...string) Result {
	t.Helper()

	cmd := exec.Command(c.BinaryPath, args...)
	cmd.Dir = c.RepoRoot
	cmd.Env = append(os.Environ(), "PATH="+c.ToolBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd.Env = append(cmd.Env, extraEnv...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	return Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
}

func DecodeJSONFile[T any](t *testing.T, path string) T {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer func() {
		_ = f.Close()
	}()

	var v T
	if err := json.NewDecoder(f).Decode(&v); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return v
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func writeStubTool(t *testing.T, dir, name, body string) {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write stub tool %s: %v", name, err)
	}
}

const fakeWP = `#!/bin/sh
set -eu

case "$*" in
  "db prefix")
    printf "wp_"
    ;;
  "option get home")
    printf "https://example.test"
    ;;
  "option get siteurl")
    printf "https://example.test"
    ;;
  "core version")
    printf "6.5.0"
    ;;
  "eval echo PHP_VERSION;")
    printf "8.2.12"
    ;;
  "eval echo is_multisite() ? \"yes\" : \"no\";")
    printf "no"
    ;;
  "post list --post_type=post --post_status=publish --format=count")
    printf "120"
    ;;
  "post list --post_type=page --post_status=publish --format=count")
    printf "12"
    ;;
  "post list --post_status=draft --format=count")
    printf "3"
    ;;
  "post list --post_status=private --format=count")
    printf "1"
    ;;
  "term list category --format=count")
    printf "8"
    ;;
  "term list post_tag --format=count")
    printf "15"
    ;;
  "user list --format=count")
    printf "4"
    ;;
  "comment list --status=approve --format=count")
    printf "22"
    ;;
  "theme list --status=active --field=name")
    printf "test-theme"
    ;;
  "plugin list --status=active --format=json")
    printf '[{"name":"advanced-custom-fields","status":"active"},{"name":"redirection","status":"active"}]'
    ;;
  "post-type list --field=name")
    printf "post\npage\nattachment\nlanding_page\n"
    ;;
  "taxonomy list --field=name")
    printf "category\npost_tag\ncampaign\n"
    ;;
  "post list --post_status=publish --post_type=post,page --fields=ID,post_type,post_name,post_title,url --format=json")
    printf '[{"ID":1,"post_type":"post","post_name":"hello","post_title":"Hello","url":"https://example.test/hello/"},{"ID":2,"post_type":"page","post_name":"about","post_title":"About","url":"https://example.test/about/"}]'
    ;;
  *)
    case "$*" in
      *"SELECT COUNT(*) FROM wp_posts WHERE post_content LIKE '%wp-content/uploads%'"*)
        printf "7"
        ;;
      *"SELECT COUNT(*) FROM wp_posts WHERE post_content LIKE '%http://%'"*)
        printf "2"
        ;;
      *"SELECT COUNT(*) FROM wp_postmeta WHERE meta_key LIKE '%yoast%' OR meta_key LIKE '%rank_math%' OR meta_key LIKE '%aioseo%'"*)
        printf "11"
        ;;
      *"SELECT COUNT(*) FROM wp_postmeta WHERE meta_value LIKE 'a:%' OR meta_value LIKE 'O:%'"*)
        printf "9"
        ;;
      *"SELECT COUNT(*) FROM wp_posts WHERE post_content REGEXP '\\[[a-zA-Z0-9_-]+'"*)
        printf "5"
        ;;
      *"SELECT post_id, meta_key, meta_value FROM wp_postmeta WHERE post_id IN"*)
        printf '1\t_yoast_wpseo_title\tHello SEO\n'
        printf '1\t_yoast_wpseo_metadesc\tThe hello description\n'
        printf '1\t_yoast_wpseo_meta-robots-noindex\t1\n'
        printf '2\trank_math_title\tAbout Page\n'
        printf '2\trank_math_description\tAbout description\n'
        ;;
      *"FROM wp_redirection_items WHERE status='enabled'"*)
        printf '/old\t/new\t301\t0\n'
        printf '^/legacy/(.*)$\t/new/$1\t302\t1\n'
        ;;
      *"FROM wp_posts p LEFT JOIN wp_postmeta pm"*)
        printf '99\t/srm-from\t/srm-to\t301\n'
        ;;
      *)
        exit 1
        ;;
    esac
    ;;
esac
`

const fakeSSH = `#!/bin/sh
set -eu

while [ "$#" -gt 0 ]; do
  case "$1" in
    -o|-p|-i)
      shift 2
      ;;
    -*)
      shift
      ;;
    *)
      shift
      break
      ;;
  esac
done

cmd="$*"
exec sh -lc "$cmd"
`

const fakePHP = `#!/bin/sh
set -eu

if [ "${1-}" != "-r" ]; then
  exit 1
fi

code="${2-}"
shift 2
if [ "${1-}" = "--" ]; then
  shift
fi

case "$code" in
  *"'base_dir' =>"* )
    dir="${1-}"
    printf '{"base_dir":"%s","total_files":1,"total_bytes":12,"extensions":{"txt":1},"files":[{"path":"2024/01/hello.txt","size":12,"ext":"txt","sha256":"","mime":"text/plain"}]}' "$dir"
    ;;
  *"'exists' =>"* )
    printf '{"exists":true,"size":12,"count":1}'
    ;;
  * )
    printf '1'
    ;;
esac
`
