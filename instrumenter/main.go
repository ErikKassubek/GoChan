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
	"runtime"
)

var path_separator string = "/"
var file_names []string = make([]string, 0)

var instrumentChan bool
var instrumentMutex bool

var in string
var out string

var show_trace bool

// read command line arguments
func command_line_args() error {
	instrumentChan_ := flag.Bool("chan", false, "instrument for channel")
	instrumentMutex_ := flag.Bool("mut", false, "instrument for mutex")
	flag.StringVar(&in, "in", "", "input path")
	flag.StringVar(&out, "out", "."+path_separator+"output"+path_separator, "output path")
	show_trace_ := flag.Bool("show_trace", false, "show the trace")

	flag.Parse()

	instrumentChan = *instrumentChan_
	instrumentMutex = *instrumentMutex_
	show_trace = *show_trace_

	if in == "" {
		return errors.New("flag -in missing or incorrect.\n" +
			"usage: go run main.go -in=[pathToFiles] <-out=[path_to_folder]> <-print_trace>)")
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
	err = getAllFiles()
	if err != nil {
		panic(err)
	}

	// instrument all files in file_names
	err = instrument_files()
	if err != nil {
		panic(err)
	}
}
