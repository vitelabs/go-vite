---
order: 4
---
# VEP 6: Vite URI Formatting

## Introduction
This specification defines standard URI format used in Vite

## Syntax
```c++
request                 = "vite" ":"[target_address] [ "@" chain_id ] [ "/" function_name ] [ "?" parameters ]
target_address          = vite_address
chain_id                = 1*DIGIT
function_name           = STRING
vite_address            = ( "vite_" 50*HEXDIG ) / VNS_NAME
parameters              = parameter *( "&" parameter )
parameter               = key "=" value
key                     = "tti" / "amount" / "fee" / "data"
value                   = number / vite_address / token_type_id / STRING
token_type_id           = "tti_" 24 *HEXDIG
number                  = [ "-" / "+" ] *DIGIT [ "." 1*DIGIT ] [ ( "e" / "E" ) [ 1*DIGIT ]
```

***STRING*** - A `URLEncode` string, using ***%*** to escape all of delimiters. Delimiters that need to be escaped include ***@/?&=%:***

***number*** - A number represented in scientific notation

***VNS_NAME*** - Vite name service. Specific standard is being worked out

### Syntax

| Field | Description |
| --- | --- |
| target_address | It indicates transfer address when transferring and refers to contract address when calling smart contract |
| chain_id | Identify MainNet, TestNet and private net. This part can be omitted, and then use current network type of client side |
| function_name | Function name of calling smart contract, if it's contract calling, there must be '/' mark, if there isn't, the calling is a common transfer. function_name can be an empty string, which indicates default contract method of current address. Attention: currently when we call smart contract, contract method signature and parameter information can be represented by setting data field, thus function_name do not do any analyse work, and you can call smart contract by specifying function_name and contract parameters by then. |
| parameters | params, key including "tti" / "amount" / "fee" / "data" |

### Parameters

| key | value | Description | Example |
| --- | --- | --- | --- |
| tti | token_type_id | Specify transfer Token Id, optional. If it's omitted that means Vite Token is making the transfer. | tti=tti_5649544520544f4b454e6e40 |
| amount | number | Specify transfer amount, unit follows token's basic unit. For instance, transfer 1 VITE equals amount = 1, optional. If it's omitted means amount = 0 | amount=1e-3，amount=1000，amount=0.04 |
| fee | number | Specify Vite volume that need to be destroyed, unit is Vite's common unit, optional. If it's omitted means fee = 0 | Same as amount |
| data | [base64 url safe encode](https://tools.ietf.org/html/rfc4648#section-5) | It means remarks when transferring, remarks need to comply with the conventional format in [VEP-8](./vep-8.md), also it represents method signature and parameter information when calling smart contract | data=MTIzYWJjZA |

## Examples

| Example | Description |
| --- | --- |
| vite:vite_fa1d81d93bcc36f234f7bccf1403924a0834609f4b2e9856ad | Represent account address vite_fa1d81d93bcc36f234f7bccf1403924a0834609f4b2e9856ad |
| vite:vite_fa1d81d93bcc36f234f7bccf1403924a0834609f4b2e9856ad?tti=tti_5649544520544f4b454e6e40&amount=1&data=MTIzYWJjZA | Transfer 1 VITE to vite_fa1d81d93bcc36f234f7bccf1403924a0834609f4b2e9856ad with a comment of “123abcd” |
| vite:vite_fa1d81d93bcc36f234f7bccf1403924a0834609f4b2e9856ad/echo?amount=1&data=MTIzYWJjZA | Call 'echo' method of contract vite_fa1d81d93bcc36f234f7bccf1403924a0834609f4b2e9856ad |
