# Go-Chan-Instrumenter

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

## What?
This program can be used to translate regular Go code 
into code usable by [GoChan](https://github.com/ErikKassubek/GoChan/tree/main/goChan).

## How to use
To use goChan, run 
```
make -IN="<input folder>" -EXEC="<executable name>"
```
input folder is the relative folder, where the code is stored.
executable name is the name of the executable produced by the code, which is translated
The translated files can be found in ./output

The program can now by run by running the main executable in the ./output
folder. 
