# Fileganizer

Fileganizer is a tool that will

- run a command to extract text from an input file,
- parse the extracted text with grok-like patterns,
- choose a pre-configured go-template depending on parsing results,
- generate a result with go-template,
- optionaly run the result (as a command).o

The use-case is to run some pdftotext command to extract text from your invoices and other similar documents, try to find patterns like IDs, date, name, and rename (move) the file using the results of the parsing.

## Tutorial

Copy `config.yaml.sample` as `config.yaml`. Edit the file:

Leave `ExtractTextCommand` as is if you have `pdftotext` installed. Or change it if you prefer using another tool.

Leave `env` as is or declare other environment variables according to your needs. These environment variables will be available in your go-templates.

Leave `commonTemplate` empty. You will fill it later, according to your needs.

Leave `months` as is or translate months into your language. This is used to convert months names into number. For example `octobre` (in French, meaning `october` can be converted to `10`).

Leave `grokPatterns` as is. You may add new patterns later, according to your needs.

Now we will work with `fileDescriptions` that contains `patterns` to try to apply on the input file and `output` as a go-template that we configure as a shell command.

1. Run `fileganizer -c config.yaml -f yourfile.pdf -t`. This will print the output of the `ExtractTextCommand`.
2. identify some interesting patterns, for example a date, an identifier...
3. add these patterns with grok syntax (learn with [Grok filter plugin from Logstash](https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html)). Note that the parser is [Grokky](https://github.com/logrusorgru/grokky) and is not fully compatible with Grok.
4. forge a go-template output with all avaiable variables (`.filename`, `.env.XXX` for environment variables, `.grok.xxx` for parsed data.
5. Run `fileganizer -c config.yaml -f yourfile.pdf` (without the `-t` option). This do all the job and print the generated result.

You can iterate as many times as you need to improve the template. You can also add other `fileDescriptions` to identify other document types and print from other go-templates.

When you want to run the output as a shell command, add `-r` option: `fileganizer -c config.yaml -f yourfile.pdf -r`.

## Build

```
go build
```

## Run

Run `fileganizer` on a file and print the generated output:
```
./fileganizer -c <config.yaml> -f <file.pdf>
```

Run `fileganizer` on a file and run the generated output:
```
./fileganizer -c <config.yaml> -f <file.pdf> -r
```

Show pdf text contents
```
./fileganizer -c <config.yaml> -f <file.pdf> -t
```

## Licensing

This project is licensed under the MIT License. See the LICENSE file for the full license text.
