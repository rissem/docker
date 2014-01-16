package docker

import (
	"github.com/dotcloud/docker"
	"github.com/dotcloud/docker/engine"
	"strings"
	"testing"
	"time"
)

func TestImageTagImageDelete(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	srv := mkServerFromEngine(eng, t)

	initialImages := getAllImages(eng, t)
	if err := eng.Job("tag", unitTestImageName, "utest", "tag1").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest/docker", "tag2").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest:5000/docker", "tag3").Run(); err != nil {
		t.Fatal(err)
	}

	images := getAllImages(eng, t)

	nExpected := len(initialImages.Data[0].GetList("RepoTags")) + 3
	nActual := len(images.Data[0].GetList("RepoTags"))
	if nExpected != nActual {
		t.Errorf("Expected %d images, %d found", nExpected, nActual)
	}

	if _, err := srv.DeleteImage("utest/docker:tag2", true); err != nil {
		t.Fatal(err)
	}

	images = getAllImages(eng, t)

	nExpected = len(initialImages.Data[0].GetList("RepoTags")) + 2
	nActual = len(images.Data[0].GetList("RepoTags"))
	if nExpected != nActual {
		t.Errorf("Expected %d images, %d found", nExpected, nActual)
	}

	if _, err := srv.DeleteImage("utest:5000/docker:tag3", true); err != nil {
		t.Fatal(err)
	}

	images = getAllImages(eng, t)

	nExpected = len(initialImages.Data[0].GetList("RepoTags")) + 1
	nActual = len(images.Data[0].GetList("RepoTags"))

	if _, err := srv.DeleteImage("utest:tag1", true); err != nil {
		t.Fatal(err)
	}

	images = getAllImages(eng, t)

	if images.Len() != initialImages.Len() {
		t.Errorf("Expected %d image, %d found", initialImages.Len(), images.Len())
	}
}

func TestCreateRm(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, _, _, err := docker.ParseRun([]string{unitTestImageID, "echo test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	job := eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err := job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 1 {
		t.Errorf("Expected 1 container, %v found", len(outs.Data))
	}

	job = eng.Job("container_delete", id)
	job.SetenvBool("removeVolume", true)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err = job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 0 {
		t.Errorf("Expected 0 container, %v found", len(outs.Data))
	}

}

func TestCreateRmVolumes(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, hostConfig, _, err := docker.ParseRun([]string{"-v", "/srv", unitTestImageID, "echo", "test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	job := eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err := job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 1 {
		t.Errorf("Expected 1 container, %v found", len(outs.Data))
	}

	job = eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("stop", id)
	job.SetenvInt("t", 1)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("container_delete", id)
	job.SetenvBool("removeVolume", true)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err = job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 0 {
		t.Errorf("Expected 0 container, %v found", len(outs.Data))
	}
}

func TestCommit(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, _, _, err := docker.ParseRun([]string{unitTestImageID, "/bin/cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	job := eng.Job("commit", id)
	job.Setenv("repo", "testrepo")
	job.Setenv("tag", "testtag")
	job.SetenvJson("config", config)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}
}

func TestRestartKillWait(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	runtime := mkRuntimeFromEngine(eng, t)
	defer runtime.Nuke()

	config, hostConfig, _, err := docker.ParseRun([]string{"-i", unitTestImageID, "/bin/cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	job := eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err := job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 1 {
		t.Errorf("Expected 1 container, %v found", len(outs.Data))
	}

	job = eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}
	job = eng.Job("kill", id)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	eng, err = engine.New(eng.Root())
	if err != nil {
		t.Fatal(err)
	}

	job = eng.Job("initapi")
	job.Setenv("Root", eng.Root())
	job.SetenvBool("AutoRestart", false)
	// TestGetEnabledCors and TestOptionsRoute require EnableCors=true
	job.SetenvBool("EnableCors", true)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	srv = mkServerFromEngine(eng, t)

	job = srv.Eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err = job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 1 {
		t.Errorf("Expected 1 container, %v found", len(outs.Data))
	}

	setTimeout(t, "Waiting on stopped container timedout", 5*time.Second, func() {
		job = srv.Eng.Job("wait", outs.Data[0].Get("ID"))
		var statusStr string
		job.Stdout.AddString(&statusStr)
		if err := job.Run(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestCreateStartRestartStopStartKillRm(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	config, hostConfig, _, err := docker.ParseRun([]string{"-i", unitTestImageID, "/bin/cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createTestContainer(eng, config, t)

	job := srv.Eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err := job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 1 {
		t.Errorf("Expected 1 container, %v found", len(outs.Data))
	}

	job = eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("restart", id)
	job.SetenvInt("t", 15)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("stop", id)
	job.SetenvInt("t", 15)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("start", id)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("kill", id).Run(); err != nil {
		t.Fatal(err)
	}

	// FIXME: this failed once with a race condition ("Unable to remove filesystem for xxx: directory not empty")
	job = eng.Job("container_delete", id)
	job.SetenvBool("removeVolume", true)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	job = srv.Eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err = job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 0 {
		t.Errorf("Expected 0 container, %v found", len(outs.Data))
	}
}

func TestRunWithTooLowMemoryLimit(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	// Try to create a container with a memory limit of 1 byte less than the minimum allowed limit.
	job := eng.Job("create")
	job.Setenv("Image", unitTestImageID)
	job.Setenv("Memory", "524287")
	job.Setenv("CpuShares", "1000")
	job.SetenvList("Cmd", []string{"/bin/cat"})
	var id string
	job.Stdout.AddString(&id)
	if err := job.Run(); err == nil {
		t.Errorf("Memory limit is smaller than the allowed limit. Container creation should've failed!")
	}
}

func TestRmi(t *testing.T) {
	eng := NewTestEngine(t)
	srv := mkServerFromEngine(eng, t)
	defer mkRuntimeFromEngine(eng, t).Nuke()

	initialImages := getAllImages(eng, t)

	config, hostConfig, _, err := docker.ParseRun([]string{unitTestImageID, "echo", "test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	containerID := createTestContainer(eng, config, t)

	//To remove
	job := eng.Job("start", containerID)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("wait", containerID).Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("commit", containerID)
	job.Setenv("repo", "test")
	var imageID string
	job.Stdout.AddString(&imageID)
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", imageID, "test", "0.1").Run(); err != nil {
		t.Fatal(err)
	}

	containerID = createTestContainer(eng, config, t)

	//To remove
	job = eng.Job("start", containerID)
	if err := job.ImportEnv(hostConfig); err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("wait", containerID).Run(); err != nil {
		t.Fatal(err)
	}

	job = eng.Job("commit", containerID)
	job.Setenv("repo", "test")
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	images := getAllImages(eng, t)

	if images.Len()-initialImages.Len() != 2 {
		t.Fatalf("Expected 2 new images, found %d.", images.Len()-initialImages.Len())
	}

	_, err = srv.DeleteImage(imageID, true)
	if err != nil {
		t.Fatal(err)
	}

	images = getAllImages(eng, t)

	if images.Len()-initialImages.Len() != 1 {
		t.Fatalf("Expected 1 new image, found %d.", images.Len()-initialImages.Len())
	}

	for _, image := range images.Data {
		if strings.Contains(unitTestImageID, image.Get("ID")) {
			continue
		}
		if image.GetList("RepoTags")[0] == "<none>:<none>" {
			t.Fatalf("Expected tagged image, got untagged one.")
		}
	}
}

func TestImagesFilter(t *testing.T) {
	eng := NewTestEngine(t)
	defer nuke(mkRuntimeFromEngine(eng, t))

	if err := eng.Job("tag", unitTestImageName, "utest", "tag1").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest/docker", "tag2").Run(); err != nil {
		t.Fatal(err)
	}

	if err := eng.Job("tag", unitTestImageName, "utest:5000/docker", "tag3").Run(); err != nil {
		t.Fatal(err)
	}

	images := getImages(eng, t, false, "utest*/*")

	if len(images.Data[0].GetList("RepoTags")) != 2 {
		t.Fatal("incorrect number of matches returned")
	}

	images = getImages(eng, t, false, "utest")

	if len(images.Data[0].GetList("RepoTags")) != 1 {
		t.Fatal("incorrect number of matches returned")
	}

	images = getImages(eng, t, false, "utest*")

	if len(images.Data[0].GetList("RepoTags")) != 1 {
		t.Fatal("incorrect number of matches returned")
	}

	images = getImages(eng, t, false, "*5000*/*")

	if len(images.Data[0].GetList("RepoTags")) != 1 {
		t.Fatal("incorrect number of matches returned")
	}
}

func TestImageInsert(t *testing.T) {
	eng := NewTestEngine(t)
	defer mkRuntimeFromEngine(eng, t).Nuke()
	srv := mkServerFromEngine(eng, t)

	// bad image name fails
	if err := srv.Eng.Job("insert", "foo", "https://www.docker.io/static/img/docker-top-logo.png", "/foo").Run(); err == nil {
		t.Fatal("expected an error and got none")
	}

	// bad url fails
	if err := srv.Eng.Job("insert", unitTestImageID, "http://bad_host_name_that_will_totally_fail.com/", "/foo").Run(); err == nil {
		t.Fatal("expected an error and got none")
	}

	// success returns nil
	if err := srv.Eng.Job("insert", unitTestImageID, "https://www.docker.io/static/img/docker-top-logo.png", "/foo").Run(); err != nil {
		t.Fatalf("expected no error, but got %v", err)
	}
}

// Regression test for being able to untag an image with an existing
// container
func TestDeleteTagWithExistingContainers(t *testing.T) {
	eng := NewTestEngine(t)
	defer nuke(mkRuntimeFromEngine(eng, t))

	srv := mkServerFromEngine(eng, t)

	// Tag the image
	if err := eng.Job("tag", unitTestImageID, "utest", "tag1").Run(); err != nil {
		t.Fatal(err)
	}

	// Create a container from the image
	config, _, _, err := docker.ParseRun([]string{unitTestImageID, "echo test"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	id := createNamedTestContainer(eng, config, t, "testingtags")
	if id == "" {
		t.Fatal("No id returned")
	}

	job := srv.Eng.Job("containers")
	job.SetenvBool("all", true)
	outs, err := job.Stdout.AddListTable()
	if err != nil {
		t.Fatal(err)
	}
	if err := job.Run(); err != nil {
		t.Fatal(err)
	}

	if len(outs.Data) != 1 {
		t.Fatalf("Expected 1 container got %d", len(outs.Data))
	}

	// Try to remove the tag
	imgs, err := srv.DeleteImage("utest:tag1", true)
	if err != nil {
		t.Fatal(err)
	}

	if len(imgs.Data) != 1 {
		t.Fatalf("Should only have deleted one untag %d", len(imgs.Data))
	}

	if untag := imgs.Data[0].Get("Untagged"); untag != unitTestImageID {
		t.Fatalf("Expected %s got %s", unitTestImageID, untag)
	}
}
