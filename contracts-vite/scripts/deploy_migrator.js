// scripts/deploy.js
require('@openzeppelin/hardhat-upgrades');

async function main() {
    const Migrator = await ethers.getContractFactory("ViteMigrator");
    console.log("Deploying Migrator ...");
    const migrator = await upgrades.deployProxy(Migrator, ['0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266', '0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266'], { initializer: 'initialize' });
    await migrator.deployed();
    console.log("deployed to:", migrator.address);
}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });
