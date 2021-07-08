// contracts/ViteToken.sol
// SPDX-License-Identifier: GPL-3.03
pragma solidity ^0.6.0;

// @openzeppelin/contracts@v3.3.0
import "@openzeppelin/contracts/math/SafeMath.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/presets/ERC20PresetMinterPauser.sol";

contract ViteToken is ERC20PresetMinterPauser {
    uint256 private erc20_decimals = 18;
    uint256 private erc20_units = 10**erc20_decimals;

    constructor() public ERC20PresetMinterPauser("Vite Token", "VITE") {}

    /**
     * @dev See {ERC20-_beforeTokenTransfer}.
     */
    function _beforeTokenTransfer(
        address from,
        address to,
        uint256 amount
    ) internal override(ERC20PresetMinterPauser) {
        super._beforeTokenTransfer(from, to, amount);
    }
}
