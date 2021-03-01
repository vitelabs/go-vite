// scripts/deploy.js
async function main() {
    const ERC20 = await ethers.getContractFactory("ViteToken");
    console.log("Deploying ERC20 ...");
    const erc20 = await ERC20.deploy();
    await erc20.deployed();
    console.log("deployed to:", erc20.address);
}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });
