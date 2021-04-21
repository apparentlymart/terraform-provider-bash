# `bash_script` example

This directory contains a simple but contrived Terraform configuration that
uses the `bash_script` data source to generate a file `example.sh`, which you
can then execute using Bash to see how it makes use of some data that
originated in the Terraform configuration.

This example includes a string, an integer, an indexed array, and an
associative array example, and so the source script template (in
`example.sh.tmpl`) also doubles as a simple example of some common ways to
interact with those data types within the Bash language.
