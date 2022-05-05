## Generating test certificates
You can generate new test certificates by running the script `self-signed.sh` located within the `./scripts/certs/` folder.

You can provide it a folder to output the cert/key to.

```shell
$ cd ./scripts/certs

$ ./self-signed.sh test-folder

$ ./self-signed.sh test
test
Generating public/private rsa key pair.
Your identification has been saved in test/key.
Your public key has been saved in test/key.pub.
The key fingerprint is:
SHA256:E9qGCsn3RVENfgrv52i1x+Ue0DBuuf2JNFEZhidASig gao@Stevens-MacBook-Pro.local
The key's randomart image is:
+---[RSA 2048]----+
|        oo+=. .o |
|     E ..o. .o..o|
|      . +.. .ooo |
| . .   = + o. *  |
|  + . o S o  * . |
|   o o o o  o = .|
|    . .   ...* = |
|          .+o = =|
|         .. .o oo|
+----[SHA256]-----+

$ ls -lah test
total 24
drwxr-xr-x  5 gao  staff   160B 19 Jan 17:27 .
drwxr-xr-x  6 gao  staff   192B 19 Jan 17:27 ..
-rw-r--r--  1 gao  staff   1.4K 19 Jan 17:27 cert
-rw-------  1 gao  staff   1.6K 19 Jan 17:27 key
-rw-r--r--  1 gao  staff   411B 19 Jan 17:27 key.pub
```
