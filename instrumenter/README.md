# Go-Chan-Instrumenter

```diff 
- This code is still work in progress and may not work or result in incorrect behavior!
```

## What?
This program can be used to translate regular Go code 
into code usable by the [GoChan-Tracer](https://github.com/ErikKassubek/GoChan/tree/main/tracer).

## How to use
To use the tracer, run 
```
cd instrumenter
go build
cd ..

./instrumenter/instrumenter -in=[input_folder] <-out=[outout_folder]> <-show_trace>
```
The tags -out and -show trace are not mandatory.
The translated files can be found in [output_folder] or ./output, if out was not specified.

To use the tracer, 
```
go get github.com/ErikKassubek/GoChan/tracer
```
must be installed on the output files.