# gencsv

Generate CSV files with fake data using [gofakeit](https://github.com/brianvoe/gofakeit).

## Quick Start

```bash
cd tools/gencsv
go run . -n 100 -h "email:Email" data.csv

# moore 
go run . -n 1000000 -h "first:FirstName" -h "last:LastName" -h "email:Email" output.csv
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-n <int>` | `1` | Total rows to generate |
| `-h`, `-H`, `--header` | _(required)_ | Column as `name:Type` (repeatable) |
| `[output.csv]` | `output.csv` | Output file path (last positional arg) |

## Types

| Type | Description |
|------|-------------|
| `FirstName` | Random first name |
| `LastName` | Random last name |
| `Email` | Random email address |
| `Phone` | Random phone number |

## Examples

Single column:

```bash
go run . -n 100 -h "email:Email" emails.csv
```

Multiple columns:

```bash
go run . -n 1000 \
  -h "first:FirstName" \
  -h "last:LastName" \
  -h "email:Email" \
  -h "phone:Phone" \
  users.csv
```

Using `-H` shorthand:

```bash
go run . -n 5 -H "name:FirstName" -H "email:Email" contacts.csv
```

Default output path:

```bash
go run . -n 10 -h "email:Email"
# writes to output.csv
```

## Build

```bash
cd tools/gencsv
go build -o gencsv .
./gencsv -n 100 -h "email:Email" data.csv
```
