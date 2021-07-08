const TokenContract = artifacts.require("CoinToken");

module.exports = function (deployer) {
  deployer.deploy(TokenContract, "Vite", "VITE", 18, 1, "0x047F2837372358bD867DB94D0f0E93388004d0Cc");
};