// test/erc20_migrator_test.js
// Load dependencies
const { use, expect, assert } = require('chai');
const { solidity } = require('ethereum-waffle');
const { ethers } = require('hardhat');
const keccak256 = require('keccak256')

use(solidity);

let migrator;
let token1;
let token2;

async function deployContract(name) {
  let contractMode = await ethers.getContractFactory(name);
  let contractIns = await contractMode.deploy();
  await contractIns.deployed();
  return contractIns;
}

// Start test block
describe('ERC20 Migrator', function () {
  beforeEach(async function () {
    token1 = await deployContract('ViteToken');
    token2 = await deployContract('ViteToken');

    migrator = await (await ethers.getContractFactory('ViteMigrator')).deploy(token1.address, token2.address);
    await migrator.deployed();
  });

  // Test case
  it('verify migrator from and to address', async function () {
    const from = await migrator.from();
    const to = await migrator.to();
    console.log(from.toString());
    console.log(to.toString());
    assert.equal(from.toString(), token1.address, 'from address');
    assert.equal(to.toString(), token2.address, 'to address');
  });


  it('erc20 migration', async function () {
    const amount = 10 ** 18;
    const [account1] = await ethers.getSigners();

    // mint
    await token1.mint(account1.address, amount.toString());
    assert.equal((await token1.balanceOf(account1.address)).toString(), amount.toString(), 'before migration');

    // grant minter role
    await token2.grantRole(keccak256('MINTER_ROLE'), migrator.address);

    await expect(
      migrator.migrate(amount.toString())
    ).to.be.revertedWith('revert ERC20: transfer amount exceeds allowance');

    // migrate from to
    await token1.approve(migrator.address, amount.toString());
    await migrator.migrate(amount.toString());

    assert.equal((await token2.balanceOf(account1.address)).toString(), amount.toString(), 'after migration');
    assert.equal((await token1.balanceOf(account1.address)).toString(), '0', 'after migration');
  });


});
