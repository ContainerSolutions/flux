package release

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ContainerSolutions/flux/git"
	"github.com/ContainerSolutions/flux/instance"
	"github.com/ContainerSolutions/flux/platform/kubernetes/testdata"
)

func TestCloneCommitAndPush(t *testing.T) {
	r, cleanup := setupRepo(t)
	defer cleanup()
	inst := &instance.Instance{Repo: r}
	ctx := NewReleaseContext(inst)
	defer ctx.Clean()

	if err := ctx.CloneRepo(); err != nil {
		t.Fatal(err)
	}

	err := ctx.CommitAndPush("No changes!")
	if err != git.ErrNoChanges {
		t.Errorf("expected ErrNoChanges, got %s", err)
	}

	// change a file and try again
	for name, _ := range testdata.Files {
		if err = execCommand("rm", filepath.Join(ctx.WorkingDir, name)); err != nil {
			t.Fatal(err)
		}
		break
	}
	err = ctx.CommitAndPush("Removed file")
	if err != nil {
		t.Fatal(err)
	}
}

func setupRepo(t *testing.T) (git.Repo, func()) {
	newDir, cleanup := testdata.TempDir(t)

	filesDir := filepath.Join(newDir, "files")
	gitDir := filepath.Join(newDir, "git")
	if err := execCommand("mkdir", filesDir); err != nil {
		t.Fatal(err)
	}

	var err error
	if err = execCommand("git", "-C", filesDir, "init"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = testdata.WriteTestFiles(filesDir); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = execCommand("git", "-C", filesDir, "add", "--all"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = execCommand("git", "-C", filesDir, "commit", "-m", "'Initial revision'"); err != nil {
		cleanup()
		t.Fatal(err)
	}

	if err = execCommand("git", "clone", "--bare", filesDir, gitDir); err != nil {
		t.Fatal(err)
	}

	return git.Repo{
		URL:    gitDir,
		Branch: "master",
	}, cleanup
}

func execCommand(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	fmt.Printf("exec: %s %s\n", cmd, strings.Join(args, " "))
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	return c.Run()
}
