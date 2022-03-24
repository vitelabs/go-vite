#!/bin/bash
# Exit immediately if a command exits with a non-zero status.
set -e

SCRIPT_DIR=$(
	cd $(dirname ${BASH_SOURCE[0]})
	pwd
)
echo $SCRIPT_DIR

# Set go-vite base directory
BASE_DIR=$(echo $SCRIPT_DIR | awk -F 'go-vite' '{print $1"go-vite"}')
echo $BASE_DIR

cd $BASE_DIR

BIN_DIR=~/go/bin/goplantuml
JAR_DIR=~/Downloads/plantuml.jar
DIAGRAMS_DIR=$BASE_DIR/docs/images/summary_diagrams/puml

# rm -rf $DIAGRAMS_DIR

mkdir -p $DIAGRAMS_DIR

# DIR_BLACKLIST=(bin build client cmd common conf contracts-vite crypto docker docs interfaces ledger log15 monitor net node pow producer rpc rpcapi smart-contract tools version vm vm_db wallet)
DIR_BLACKLIST=(bin build conf contracts-vite docker docs smart-contract version)

for d in $BASE_DIR/*; do
	# Skip non-directories
	! [ -d "$d" ] && continue
	dir_name=${d#"$BASE_DIR/"}
	# Skip blacklisted dir names
	[[ " ${DIR_BLACKLIST[*]} " =~ " ${dir_name} " ]] && continue
	# Skip if already exists
	[ -f "$DIAGRAMS_DIR/${dir_name}.puml" ] && continue
	echo $dir_name
	$BIN_DIR -recursive ./${dir_name} > $DIAGRAMS_DIR/${dir_name}_full.puml
	$BIN_DIR -recursive -hide-fields -hide-methods ./${dir_name} > $DIAGRAMS_DIR/${dir_name}.puml
	# Special case: replace ""net.Conn with "net.Conn
	sed -i 's/""net.Conn/"net.Conn/' $DIAGRAMS_DIR/${dir_name}_full.puml
	sed -i 's/""net.Conn/"net.Conn/' $DIAGRAMS_DIR/${dir_name}.puml
done

# $BIN_DIR -recursive -hide-fields -hide-methods ./client > $DIAGRAMS_DIR/client.puml

for puml_file in $DIAGRAMS_DIR/*.puml; do
	file_name=${puml_file#"$DIAGRAMS_DIR/"}
	dir_name=${file_name%".puml"}
	# Skip if already exists
	[ -f "$DIAGRAMS_DIR/${dir_name}.png" ] && continue
	echo $dir_name
	# vm direcotry has largest size (94406x5318)
	java -jar $JAR_DIR -DPLANTUML_LIMIT_SIZE=95000 -verbose $puml_file
done