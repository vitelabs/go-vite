#!/bin/bash
set -e

version="$1"

if [ -z "$version" ]; then
    echo "error: upgrade_tar_gz.sh [version]"
    exit 1
fi

wget -c https://github.com/vitelabs/go-vite/releases/download/${version}/gvite-${version}-linux.tar.gz

tar xvf gvite-${version}-linux.tar.gz

echo "gvite backup."

sudo mv /usr/local/vite/gvite /usr/local/vite/gvite_`date "+%Y_%m_%d"` 2>/dev/null

echo "replace new version gvite"
sudo cp gvite-${version}-linux/gvite /usr/local/vite

echo "replace finished."

echo "restart gvite"

sudo service vite restart

echo "restart done"

/usr/local/vite/gvite version
