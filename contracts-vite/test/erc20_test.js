// test/erc20_test.js
// Load dependencies
const { use, expect, assert } = require('chai');
const { solidity } = require('ethereum-waffle');
const { ethers } = require('hardhat');
const keccak256 = require('keccak256')


const MINTER_ROLE = keccak256('MINTER_ROLE');
const ADMIN_ROLE = ethers.constants.HashZero;
use(solidity);

let token;

async function deployContract(name) {
  let contractMode = await ethers.getContractFactory(name);
  let contractIns = await contractMode.deploy();
  await contractIns.deployed();
  return contractIns;
}

// Start test block
describe('ERC20 Migrator', function () {
  beforeEach(async function () {
    token = await deployContract('ViteToken');
  });

  it('erc20 migration', async function () {
    const amount = 10 ** 18;
    const [account1, account2, account3] = await ethers.getSigners();


    await expect(
      token.connect(account2).mint(account1.address, amount.toString())
    ).to.be.revertedWith('must have minter role to mint');

    // grant minter role
    await token.grantRole(MINTER_ROLE, account2.address);

    // mint
    await token.connect(account2).mint(account1.address, amount.toString());
    assert.equal((await token.balanceOf(account1.address)).toString(), amount.toString(), 'after grant');

    // revoke minter role
    await token.revokeRole(MINTER_ROLE, account2.address);

    await expect(
      token.connect(account2).mint(account1.address, amount.toString())
    ).to.be.revertedWith('must have minter role to mint');

    // renounce role
    await token.renounceRole(MINTER_ROLE, account1.address);
    await expect(
      token.connect(account1).mint(account1.address, amount.toString())
    ).to.be.revertedWith('must have minter role to mint');

    // grant admin role
    await token.grantRole(ADMIN_ROLE, account2.address);
    await token.renounceRole(ADMIN_ROLE, account1.address);
    await expect(
      token.connect(account1).grantRole(ADMIN_ROLE, account1.address)
    ).to.be.revertedWith('sender must be an admin to grant');

    // account2 grant minter role for account1
    await token.connect(account2).grantRole(MINTER_ROLE, account1.address);
    token.connect(account1).mint(account1.address, amount.toString())
  });
});
