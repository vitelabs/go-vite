
# install && compile

```
yarn install
npx hardhat compile

```

# test

```
// run a local node
npx hardhat node 

// run the test scripts
npx hardhat test test/erc20_migrator_test.js
```

# deploy to local

```
// run a local node
npx hardhat node 

// deploy erc20 token contract
npx hardhat run --network local scripts/deploy.js 

// deploy erc20 migrator contract
npx hardhat run --network local scripts/deploy_migrator.js

```
