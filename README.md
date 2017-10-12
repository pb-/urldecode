About
=====

urldecode is a small special-purpose streaming tokenizer for application/x-www-form-urlencoded data. Its main use case is to parse very long values (tens of megabytes) without allocating memory linear in the length of the values.


Usage
-----

```go
d := urldecode.NewDecoder(req.Body()) // any io.Reader is fine
key, value, err := d.NextPair() // err is io.EOF after last pair
if err != nil {
	panic(err)
}

fmt.Printf("value of key %s: ", key)
io.Copy(os.Stdout, value) // value is an io.Reader
```


FAQ
---

**Why would anyone want to use this?**  
Unfortunately there are services out there which will happily POST 80 megabytes of data as application/x-www-form-urlencoded (hello Mandrill). This is a problem because most decoders (including the Go standard library) for this kind of data make the (sane) assumption that these data are relatively short and fit easily into memory. With this library, you can get away with a constant amount of memory and stream the data as needed.


Limitations
-----------

 * Keys (not values) are currently not being decoded, i.e., `foo%20` will not decode as `foo ` but `foo%20`. Pull requests welcome.
