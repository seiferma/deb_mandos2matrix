# Debian Repository of mandos2matrix

The repository contains deb packages for the releases of [mandos2matrix](https://github.com/seiferma/deb_mandos2matrix).
To make use of the repository, follow the steps below.

# Prerequisites
* `curl`
* `sudo`

# Installation
* Import the signing key of the repository
```sh
sudo curl -fsSL https://seiferma.github.io/deb_mandos2matrix/pubkey.gpg -o /etc/apt/keyrings/mandos2matrix.asc
```

* Add the repository to the sources of apt
```sh
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/mandos2matrix.asc] https://seiferma.github.io/deb_mandos2matrix all main" | \
  sudo tee /etc/apt/sources.list.d/mandos2matrix.list > /dev/null
```

* Update the package cache and install `mandos2matrix`
```sh
apt-get update
apt-get install mandos2matrix
```