// scripts/deploy.js
async function main() {
    const Migrator = await ethers.getContractFactory("ViteMigrator");
    console.log("Deploying Migrator ...");
    const migrator = await Migrator.deploy('0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266', '0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266');
    await migrator.deployed();
    console.log("deployed to:", migrator.address);
}

main()
    .then(() => process.exit(0))
    .catch(error => {
        console.error(error);
        process.exit(1);
    });
