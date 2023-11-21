# hasher

Hash files and verify hash.

```bash
$ hasher README.md
76cc9e9e61122aa49c5fe4ffccc491c8a4fb68acd1afb722365ed4b4623c6381  README.md
$ hasher -a blake3 README.md
17bafd514c8e8951c06949be32a6cd22b1b97638a7a0ceeb8984ca57ab32d68e  README.md
$ cat README.md | hasher --stdin
76cc9e9e61122aa49c5fe4ffccc491c8a4fb68acd1afb722365ed4b4623c6381
```
