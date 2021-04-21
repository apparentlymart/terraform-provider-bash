# Terraform Bash Provider

This is a Terraform utility provider which aims to robustly generate Bash
scripts which refer to data that originated in Terraform.

When gluing other software into Terraform it's common to generate little shell
snippets using Terraform's template language, such as when passing some shell
commands to CloudInit installed on an AWS EC2 instance using `user_data`:

```hcl
  user_data = <<-EOT
    /usr/local/bin/connect-to-something ${aws_eip.example.public_ip}
  EOT
```

This sort of thing is fine for simple cases where the generated script is
relatively simple and where the templated arguments don't need any special
escaping to be interpreted correctly by the shell, but sometimes these scripts
get more complicated and start to refer to a variety of different data from
the source Terraform configuration, which can lead to robustness issues
related to incorrect quoting/escaping and difficulty dealing with list and
map data, when relevant.

The `bash_script` data source in this provider aims to help with those more
complex cases by automatically generating properly-formatted Bash variable
declarations from a subset of Terraform language types, prepending them to
a bash script template you provide which can then make use of those variables.

```hcl
terraform {
  required_providers {
    bash = {
      source = "apparentlymart/bash"
    }
  }
}

data "bash_script" "example" {
  source = file("${path.module}/example.sh.tmpl")
  variables = {
    something_ip = aws_eip.example.public_ip
    device_names = tolist(aws_volume_attachment.example[*].device_name)
  }
}

resource "aws_instance" "example" {
  # ...
  user_data = data.bash_script.example.result
}
```

Inside `example.sh.tmpl` you can write a Bash script which assumes that
variables `something_ip` and `device_names` are predeclared:

```bash
#!/bin/bash

/usr/local/bin/connect-to-something "${something_ip}"
for device_name in "${device_names[@]}"; do
  /usr/local/bin/prepare-filesystem "/dev/${device_name}"
done
```

The `bash_script` data source will automatically generate Bash `declare`
commands to represent the `something_ip` and `device_names` variables and
then prepend that into the source script to produce a result that should work
as a self-contained bash script:

```bash
#!/bin/bash
declare -r something_ip='192.0.2.5'
declare -ra device_names=('sdb' 'sdc')

/usr/local/bin/connect-to-something "${something_ip}"
for device_name in "${device_names[@]}"; do
  /usr/local/bin/prepare-filesystem "/dev/${device_name}"
done
```

Notice that `bash_script` doesn't actually _execute_ the script you provide.
Instead, it exports a string attribute `result` which contains the script
source code, ready for you to pass to some other resource argument that expects
to recieve the source code of a Bash script.

Because `bash_script` is aware of the syntax Bash expects for strings, integers,
arrays of strings, and a associative arrays of strings, it can automatically
generate suitable quoting and other punctuation to ensure that the values
pass into Bash exactly as they appear in Terraform, without any need for
manual escaping within the Terraform template language.

All you need to do then is write a plain Bash script which uses standard Bash
language features to interact with those generated declarations. This also means
that your source script will be 100% direct Bash syntax, without any conflicts
between Terraform's interpolation syntax and Bash's interpolation syntax.

## Passing Values to Bash

Bash's type system is more limited than Terraforms, and so the entries in your
`variables` argument must each be of one of the following Terraform types:

* `string`: The most common situation, passing a single string value into the
  script, often to interpolate directly into a command line.
* `number`: Becomes an integer value in Bash, which you can then use for
  arithmetic. Bash only supports whole numbers, so you can't pass fractional
  values into your script.
* `list(string)`: Becomes an indexed array of strings in Bash. Terraform has
  a few different sequence types that can convert to a list of strings, so
  you may need to use [`tolist`](https://www.terraform.io/docs/language/functions/tolist.html)
  to ensure your value is actually a list.
* `map(string)`: Becomes an associative array of strings in Bash. Terraform has
  both object types and map types that are similar but not equivalent, so you
  may need to use [`tomap`](https://www.terraform.io/docs/language/functions/tomap.html)
  to ensure your value is actually a map.

Values of any other type in `variables` will cause an error message.

## Using Values in Bash

The `bash_script` data source ensures that all of the variables you define
will be declared correctly to avoid escaping and quoting issues, but you must
also ensure that you use those variables correctly elsewhere in the script
to avoid Bash misinterpreting how you intend the value to be used.

This can get a similar effect as interpolating literal values directly into the
generated Bash script using Terraform's template language, but with the
advantage that it's Bash itself interpreting the dynamic values, and so
in more complex scripts you can use the `if`, `case`, and `for` statements to
select different code paths depending on those values.

The following sections show some examples of common patterns that might arise
in shell scripts generated using `bash_script`. This is not a full reference on
Bash syntax though; see [the Bash Reference Manual](https://www.gnu.org/software/bash/manual/bash.html)
for all of the details.

### String Interpolation

When you refer to a string variable for interpolation, be sure to always
place the interpolation in quotes to ensure that Bash won't interpret any
spaces in the value as argument separators:

```bash
# Just a single variable alone
echo "${foo}"

# A mixture of literal characters and variables in a single pair of quotes,
# interpreted all as one argument by Bash.
echo "The value is ${foo}!"
```

If you are using a variable in the first argument position of a command, or in
some other place in a more complex command where options are expected, you may
need to take some extra care to avoid certain values being misinterpreted by
the target command as a command line option or flag. The syntax for this
varies depending on which command you are running, but a typical solution for
programs that use the GNU options style is to use the special option terminator
argument `--`, which has no direct meaning itself but forces the remaining
arguments to not be interpreted as options:

```bash
ls -- "${dir}"
```

Without the extra `--` prefix here, a `dir` value that starts with `-` would
be misinterpreted as an option rather than as a path to list.

In many situations you can alternatively write `$dir` instead of `${dir}`, with
the same effect. The braced version has the advantage that you can write other
literal tokens around it without fear that they'll be understood as part of
the interpolation. Consider that writing `$dir_foo` would be understood like
`${dir_foo}` rather than `${dir}_foo`. For that reason, it can be good to
standardize on using the braced form for human readability.

### Integer Arithmetic

When you pass a whole number into Bash, in many contexts it'll behave just like
a string containing a decimal representation of the number, but you can also
use it for arithmetic using the special `$(( ... ))` arithmetic syntax:

```bash
echo "${num} * ${num} = $(( num * num ))"
```

You can also use number values as indexes into an indexed array, as we'll see
in the next section.

### Conditional branches with `if` and `case`

Because Bash itself is interpreting the values, rather than Terraform's
template language, your script can potentially make dynamic decisions based on
the values using an `if` statement.

A simple example of this might be to take a particular action only if a
necessary variable has been set to a non-empty value:

```bash
if [ -n "${audit_host}" ]; then
  /usr/local/bin/send-audit -- ${audit_host}
fi
```

The `-n` operator tests whether the argument is a non-empty string. It's best
to always write the variable to be tested in quotes, because that ensures
the result will still be valid syntax if the variable contains spaces.

You can also test equality or inequality with a particular other value:

```bash
if [ "${validation_mode}" == "strict" ]; then
  /usr/local/bin/strict-validate
fi
```

The `==` and `!=` operators represent string equality or inequality
respectively.

A more powerful conditional statement is `case`, which allows you to
pattern-match against a value using the usual Bash "globbing" syntax:

```bash
case "${validation_mode}" in
strict)
  /usr/local/bin/strict-validate
custom-*)
  # Any string that starts with "custom-"
  /usr/local/bin/custom-validate "${validation_mode}"
*)
  # Default case for anything that doesn't match the above rules.
  >&2 echo "Invalid validation mode ${validation_mode}"
  exit 1
esac
```

### Indexing and Iterating Over Indexed Arrays

An indexed array in Bash is similar to a Terraform list in that it's an ordered
sequence of values, each of which has an index number counting up from zero.

You can access a single element of an array by providing the index in square
brackets, as in the following examples:

```bash
# A hard-coded index
echo "The first item is ${example[0]}"

# A dynamic index from a variable
echo "Item ${index} is ${example[$index]}"
```

More commonly though, we want to iterate over the elements of an array and run
one or more commands for each of them. We can do that using the Bash `for`
statement, using the special syntax `[@]` to indicate that we want to visit
one array element at a time:

```bash
for name in "${names[@]}"; do
    echo "Hello ${name}!"
done
```

Notice that again we should write the `${names[@]}` interpolation in quotes to
ensure that Bash will take each element as a single value, even if it happens
to contain spaces. The quotes here are applied to each element in turn, even
though it might seem like this would cause the entire array to be interpreted
as a single quoted value.

## Indexing and Iterating Over Associative Arrays

An associative array is similar to a Terraform map, in that it's a lookup
table of values where each value has an associated string key.

The index syntax for associative arrays is similar to indexed arrays except
that the key will be a string key instead of an integer:

```bash
# A hard-coded key
echo "The foo item is ${example["foo"]}"

# A dynamic key from another variable
echo "Item ${key} is ${example["${key}"]}"
```

We can also iterate over elements of an associative array. The same
`${array[@]}` syntax we saw for indexed arrays will work, but it'll provide
both the key and the value to each `for` iteration. If we use `${!array[@]}`
instead (note that extra exclaimation mark) then we can iterate over just
the keys, which we can in turn use with indexing to get the values:

```
for k in "${!instance_ids[@]}"; do
    echo "Instance ${k} has id ${instance_ids["$k"]}"
done
```

## The Interpreter Line

On Unix systems there is a convention that a script file may start with a
special line with the prefix `#!`, followed by another program that can
interpret the script. If you include such a line and then ensure that your
script is written with the executable permission then you can run your
script directly as a program, rather than having to pass it as an argument
to `bash` yourself:

```bash
#!/bin/bash

# (the rest of your script here)
```

Although `bash_script` typically appends your provided script to its generated
variable declarations, it has a special case to detect an interpreter line
as shown above and make sure that remains as the first line in the result,
so that you can use the resulting string as an executable script.

## Other Bash Robustness Tips

By default Bash is very liberal in how it will interpret your scripting
commands, which can make it hard to debug mistakes you might make. For example,
if you declare a variable called `foo` but make a typo as `${fo}` then by
default Bash will replace that interpolation with an empty string, rather than
returning an error.

You can override that behavior and ask Bash to generate an explicit error for
undefined references by setting the option `-x`. You can declare that within
your script by using the `set` command as one of the first commands:

```bash
set -x
```

Another common problem is that by default Bash will react to an error in an
intermediate command by continuing on regardless. That can be bothersome if
a later command relies on the result of an earlier one. You can use the
`-e` option to ask Bash to exit whenever a command encounters an error.

The `-e` option only applies to terminal commands, though. If you are using
more complex scripting features such as piping the output from one command
into another then a failure further up the pipeline will not fail the overall
pipeline by default. You can override that using the `-o pipefail` option.

Putting those all together we can make a boilerplate `set` statement that can
be useful to include in all scripts to ensure that they'll fail promptly in the
case of various common scripting mistakes:

```
set -exo pipefail
```
