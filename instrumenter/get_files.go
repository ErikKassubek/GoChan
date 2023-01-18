package main

/*
Copyright (c) 2023, Erik Kassubek
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
get_files.go
Get all files from the input path
*/

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// get all files in in and write them into file_names
// create folder structure in out
/*
Function to get all files in the in folder and add there names to file_names.
The function also copies the folder structure into the output folder.
@return []string: list of file names
@return error: Error or nil
*/
func getAllFiles() ([]string, error) {
	// remove old output folder
	err := os.RemoveAll(out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to remove old output folder %s.\n", out)
		return make([]string, 0), err
	}

	var file_names []string = make([]string, 0)

	// get all file names
	err = filepath.Walk(in,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not walk through file path %s.", path)
				return err
			}
			// only save go, mod and sum files
			if len(path) >= 4 && (path[len(path)-3:] == ".go" ||
				path[len(path)-4:] == ".mod" || path[len(path)-4:] == ".sum") {
				file_names = append(file_names, path)
			}
			return nil
		})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to walk through file path.\n")
		return make([]string, 0), err
	}

	// get folder structure in in and copy it to out
	folders := make([]string, 0)
	in_split := strings.Split(in, path_separator)
	folders = append(folders, out+in_split[len(in_split)-2])
	err = filepath.WalkDir(in,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not walk through dir path %s.\n", path)
				return err
			}

			if info.IsDir() && path[:1] != "." {
				folders = append(folders, out+path)
			}
			return nil
		})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to walk through dir path.\n")
		return make([]string, 0), err
	}

	for _, folder := range folders {
		err := os.MkdirAll(folder, os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not create folder %s.\n", folder)
			return make([]string, 0), err
		}
	}

	return file_names, nil
}
