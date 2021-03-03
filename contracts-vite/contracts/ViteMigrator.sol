// contracts/ViteMigrator.sol
// SPDX-License-Identifier: GPL-3.0
pragma solidity ^0.6.0;

// @openzeppelin/contracts@v3.3.0
import "@openzeppelin/contracts/math/SafeMath.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/SafeERC20.sol";

interface IERC20Mintable {
    /**
     * @dev See {ERC20-_mint}.
     * @dev See {ERC20PresetMinterPauser-mint}.
     *
     * Requirements:
     *
     * - the caller must have the {MINTER_ROLE}.
     */
    function mint(address account, uint256 amount) external;
}

contract ViteMigrator {
    using SafeERC20 for IERC20;

    event Migration(address indexed addr, uint256 amount);
    address public constant HOLE = 0x1111111111111111111111111111111111111111;
    address public from;
    address public to;

    constructor(address _from, address _to) public {
        from = _from;
        to = _to;
    }

    function migrate(uint256 _amount) public {
        // transfer erc20 token from
        IERC20(from).safeTransferFrom(msg.sender, HOLE, _amount);

        IERC20Mintable(to).mint(msg.sender, _amount);

        emit Migration(msg.sender, _amount);
    }
}
