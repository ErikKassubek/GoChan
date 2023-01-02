package main

/*
Copyright (c) 2022, Erik Kassubek
All rights reserved.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

/*
Author: Erik Kassubek <erik-kassubek@t-online.de>
Package: GoChan-Instrumenter
Project: Bachelor Thesis at the Albert-Ludwigs-University Freiburg,
	Institute of Computer Science: Dynamic Analysis of message passing go programs
*/

/*
main.go
main function and handling of command line arguments
*/

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
)

var path_separator string = "/"

var instrumentChan bool
var instrumentMutex bool

var in string
var out string
var execName string

var n_repl int = 0

// read command line arguments
func command_line_args() error {
	instrumentChan_ := flag.Bool("chan", false, "instrument for channel")
	instrumentMutex_ := flag.Bool("mut", false, "instrument for mutex")
	flag.StringVar(&in, "in", "", "input path")
	flag.StringVar(&out, "out", "."+path_separator+"output"+path_separator, "output path")
	flag.StringVar(&execName, "exec", "main", "name of created executable")

	flag.Parse()

	instrumentChan = *instrumentChan_
	instrumentMutex = *instrumentMutex_

	if in == "" {
		return errors.New("flag -in missing or incorrect.\n" +
			"usage: go run main.go -in=[pathToFiles] <-out=[path_to_folder]>)")
	}

	// add trailing path separator to in
	if in[len(in)-1:] != path_separator {
		in = in + path_separator
	}

	// add trailing path separator to out
	if out[len(out)-1:] != path_separator {
		out = out + path_separator
	}

	if in == out {
		return errors.New("in cannot be equal to out")
	}

	return nil
}

func main() {
	// set path separator if windows
	if runtime.GOOS == "windows" {
		path_separator = "\\"
	}

	err := command_line_args()
	if err != nil {
		panic(err)
	}

	// save all go files from in in file_names
	file_name, err := getAllFiles()
	if err != nil {
		panic(err)
	}

	// instrument all files in file_names
	err = instrument_files(file_name)
	if err != nil {
		panic(err)
	}

	// create the new main file

	// read template
	dat, err := os.ReadFile("./instrumenter/main_template.txt")
	if err != nil {
		dat, err = os.ReadFile("./main_template.txt")
		if err != nil {
			panic(err)
		}
	}
	data := string(dat)

	path := ""

	for _, f := range file_name {
		file_path_split := strings.Split(f, path_separator)
		if file_path_split[len(file_path_split)-1] == "main.go" {
			path = f
		}
	}

	if path == "" {
		panic("Could not find main file!")
	}

	path = strings.Replace(path, "main.go", execName, -1)

	// replace placeholder
	data = strings.Replace(data, "$$COMMAND$$", "./"+path, -1)

	save_size := ""
	for _, sw := range select_ops {
		save_size += "switch_size[" + fmt.Sprint(sw.id) + "] = " + fmt.Sprint(sw.size) + "\n"
	}

	data = strings.Replace(data, "$$SWITCH_SIZE$$", save_size, -1)

	f, err := os.Create(out + "main.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.Write([]byte(data))
}
