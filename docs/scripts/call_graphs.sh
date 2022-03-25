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

BIN_DIR=$SCRIPT_DIR/bin
TOOL_PATH=$BIN_DIR/callGraph/callGraph
DIAGRAMS_DIR=$BASE_DIR/docs/images/summary_diagrams/call_graphs

# Clone callGraph repository if it does not exist
if [ ! -f "$TOOL_PATH" ]; then
    mkdir -p $BIN_DIR
    cd $BIN_DIR
    git clone https://github.com/koknat/callGraph.git
fi

cd $BASE_DIR

rm -rf $DIAGRAMS_DIR

mkdir -p $DIAGRAMS_DIR

$TOOL_PATH $BASE_DIR -start 'SendRawTransaction' -output $DIAGRAMS_DIR/send_raw_transaction.png -noShow -language 'go'
# Current limitation: All methods with the same name will be included in the call graph diagram
# Temporary workaround: Rename the update function in ledger/consensus/trigger.go to ConsensusTriggerUpdate
# $TOOL_PATH $BASE_DIR -start 'ConsensusTriggerUpdate' -output $DIAGRAMS_DIR/consensus_trigger_update.png -noShow -language 'go'

# Delete .dot files
rm $DIAGRAMS_DIR/*.dot