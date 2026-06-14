pragma solidity ^0.8.30;

contract ERC20 {
    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);

    function balanceOf(address account) public view returns (uint256) {
        return 0;
    }

    function transfer(address to, uint256 value) public returns (bool) {
        return true;
    }
}

