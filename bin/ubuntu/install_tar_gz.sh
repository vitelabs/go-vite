#!/bin/bash
set -e


function isYes() {
    read -p "${1}[Y/N]" -n 1 -r
    echo    # (optional) move to a new line
    if [[ ! $REPLY =~ ^[Yy]$ ]]
    then
        exit 1
    fi
}

version="$1"

if [ -z "$version" ]; then
    echo "error: install_tar_gz.sh [version]"
    exit 1
fi

isYes "Download gvite-${version} "

echo "download gvite binary file[${version}]."

wget -c https://github.com/vitelabs/go-vite/releases/download/${version}/gvite-${version}-linux.tar.gz

tar xvf gvite-${version}-linux.tar.gz

cd gvite-${version}-linux

isYes "Install gvite-${version} "

CUR_DIR=`pwd`
CONF_DIR="/etc/vite"
BIN_DIR="/usr/local/vite"
LOG_DIR=$HOME/.gvite

echo "install config to "$CONF_DIR

sudo mkdir -p $CONF_DIR
sudo cp $CUR_DIR/node_config.json $CONF_DIR
ls  $CONF_DIR/node_config.json

echo "install executable file to "$BIN_DIR
sudo mkdir -p $BIN_DIR
mkdir -p $LOG_DIR
sudo cp $CUR_DIR/gvite $BIN_DIR

echo '#!/bin/bash
exec '$BIN_DIR/gvite' -pprof -config '$CONF_DIR/node_config.json' >> '$LOG_DIR/std.log' 2>&1' | sudo tee $BIN_DIR/gvited > /dev/null

sudo chmod +x $BIN_DIR/gvited

ls  $BIN_DIR/gvite
ls  $BIN_DIR/gvited

echo "config vite service boot."

echo '[Unit]
Description=GVite node service
After=network.target

[Service]
ExecStart='$BIN_DIR/gvited'
Restart=on-failure
User='`whoami`'
LimitCORE=infinity
LimitNOFILE=10240
LimitNPROC=10240

[Install]
WantedBy=multi-user.target' | sudo tee /etc/systemd/system/vite.service>/dev/null

sudo systemctl daemon-reload

# sudo systemctl enable vite
