caldr
-----

![caldr](caldr.png)

`caldr`, the command line **cal**en**d**a**r**. It's super lightweight, yet it
supports CardDAV sync!

## Build

```sh
go build .
```

## Usage

Either export all necessary variables to your ENV or set them as command line
flags:

```sh
export CARDDAV_USERNAME='...'
export CARDDAV_PASSWORD='...'
export CARDDAV_ENDPOINT='...'
export CALDR_DB='...'
```

If you're using [Baïkal](https://github.com/sabre-io/Baikal) for example, you
would export something like this as `CARDDAV_ENDPOINT`:

```sh
export CARDDAV_ENDPOINT='https://my.baik.al/dav.php/'
```

The `CALDR_DB` is the local events database in order to not need to contact
the CalDAV for every lookup. You might set it to something like this:

```sh
export CALDR_DB=~/.cache/caldr.db
```

When `caldr` is launched for the first time, it requires the `-r` flag to
refresh the events and sync them locally: 

```sh
caldr -r
```

This way you could create a cron job that refreshes `caldr` in the background,
e.g. every three hours:

```sh
crontab -e
```

```crontab
0 */3 * * * sh -c 'caldr -r'
```

You can also output contacts as JSON format using the `-j` flag:

```sh
caldr -j
```

Find more flags and info with `caldr --help`.


## FAQ

- Q: Does `caldr` write/modify any contact information?
  A: Nope, so far it's read-only and does not support updating iCals, hence it
     won't mess with your data.
- Q: Can I use it with my local calendar?
  A: Nope, as of right now `caldr` only supports CalDAV servers to sync with.
- Q: Does it support HTTP Digest auth?
  A: Nope, only HTTP Basic auth.

[1]: https://pkg.go.dev/text/template
[2]: example.tmpl
[3]: https://pkg.go.dev/github.com/emersion/go-vcard#Card
[4]: https://stedolan.github.io/jq/

