package app

import "log"

// import (
// 	"io"
// 	"log"
// 	"os"
// 	"os/exec"
// )

var noForwarder = Forwarder{}

// var chanFwd = make(chan Forwarder)

type Forwarder struct {
	exec   string
	config string
}

func setForwarder(fwd Forwarder) {
	// TODO: implement forwarder handling
	log.Printf("%#v", fwd)
	// chanFwd <- fwd
}

/*
var fwdDir = ".forwader"

func forwarder() {
	var cmd *exec.Cmd
	var err error

	if err = os.Mkdir(fwdDir, 0700); err != nil && !os.IsExist(err) {

		log.Fatalf("Err Can not created forwarder dir %q: %v", fwdDir, err)
	}

	chanTerminate := make(chan error)
	fwd := <-chanFwd

	for true {
		if fwd != noForwarder {
			log.Printf("Forwarder: %s (%s)", fwd.exec, fwd.config)

			if fwd.config != "" {
				if err = copy(fwd.config, fwdDir+"/global_config.json"); err != nil {
					log.Fatalf("Err Can not copy forwarder config %q: %v", fwd.config, err)
				}
			}

			cmd = exec.Command(fwd.exec)
			cmd.Dir = fwdDir
			if err = cmd.Start(); err != nil {
				log.Fatalf("Err Can not start forwarder %q: %v", fwd.exec, err)
			}
			go waitTerminate(cmd, chanTerminate)
		}

		select {
		case fwd = <-chanFwd:
			log.Println("Changing forwarder, killing old forwarder ...")
			cmd.Process.Kill()
			<-chanTerminate
		case err = <-chanTerminate:
			log.Printf("Err Forwarder terminated: %v", err)
			fwd = <-chanFwd
		}
	}
}

func waitTerminate(cmd *exec.Cmd, chanTerminate chan error) {
	chanTerminate <- cmd.Wait()
}

func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}
/**/
