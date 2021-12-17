---
order: 2
---

# ViteX Gateway Protocol

:::tip Introduction
This specification describes the API interface that a third-party ViteX gateway need to implement for integrating into ViteX. 
It has been fully supported by [Vite Web Wallet](https://github.com/vitelabs/vite-web-wallet) and Vite Mobile App.

For how to integrate the gateway in the wallet, please see [Gateway Integration Guide](./integration-guide.md)
:::

:::warning Attention
* HTTPS is required
* CORS (Cross-Origin Resource Sharing) must be supported
* Transfer amount should be expressed in minimum precision
:::

## Request

### header
The following parameter(s) are appended in request header from Vite web wallet
  |Name|Description|
  |:--|:--|
  |lang|The wallet will pass the current locale, and the gateway should handle it to provide i18n support.<br>`zh-cn`(Chinese simplified) and `en`(English) are currently supported in the wallet.|
  |version|The spec version number(s) currently supported by the web wallet, split by `,`|
  
## Response

### header
The following parameter(s) should be appended in response header from gateway
  |Name|Description|
  |:--|:--|
  |version|The spec version number(s) currently supported by the gateway, split by `,`|
  
### body
```javascript
{
  "code": 0,//response code
  "subCode": 0,//sub code filled in by gateway for debugging
  "msg": null,//additional message filled in by gateway for debugging
  "data":""//response data
}
```
## Metadata API

### `/meta-info`

Get gateway information by token id

* **Method**: `GET`

* **Request**: `query string`

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |tokenId|Token id|string|true|
  
* **Response**

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |type|Binding type. Allowed value: <br>`0`Independent address mode<br>`1`Bind-by-comment mode|int|true|
  |depositState|Deposit channel state. Allowed value: `OPEN`, `MAINTAIN`, `CLOSED`|string|true|
  |withdrawState|Withdrawal channel state. Allowed value: `OPEN`, `MAINTAIN`, `CLOSED`|string|true|

:::tip Binding Types
The web wallet needs use binding types to render different deposit/withdrawal UI(s) and build different requests, while the gateway needs use it to return different responses. 
At the time being the following two types have been defined:
* `0` Independent address mode: In this mode, the gateway will bind a separate inbound address to each user's Vite address. Examples of this type are BTC and ETH<br>
* `1` Bind-by-comment mode: In this mode, the gateway cannot bind separate inbound address to each user's Vite address, so that it is necessary to identify the user's VITE address with additional comment. Examples of this type are EOS and XMR
:::

* **Example**

  ***Request***
    ```
    /meta-info?tokenId=tti_82b16ac306f0d98bf9ccf7e7
    ```
  ***Response***
    ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "type": 0,
        "depositState": "OPEN",
        "withdrawState": "OPEN"
      }
    }
    ```

## Deposit/withdrawal API

### `/deposit-info`

Get deposit information by token id and user's Vite address. 
The gateway should bind user's Vite address to a source chain address, and the web wallet will display a deposit page based on the API response.

* **Method**: `GET`

* **Request**: `query string`

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |tokenId|Gateway token id|string|true|
  |walletAddress|User's Vite address|string|true|
  
* **Response**

	|Name|Description|Data Type|Required|
	|:--|:---|:---:|:---:|
	|depositAddress|Deposit address|string|true|
	|labelName|Label name, required if type=1|string|false|
	|label|Label value, required if type=1|string|false|
	|minimumDepositAmount|Minimum deposit amount|string|true|
	|confirmationCount|Confirmations on source chain|int|true|
	|noticeMsg|Extra message filled in by gateway|string|false|

* **Example**

  ***Request***
    ```
    /deposit-info?tokenId=tti_82b16ac306f0d98bf9ccf7e7&walletAddress=vite_52ea0d88812350817df9fb415443f865e5cf4d3fddc9931dd9
    ```
  ***Response***
  
  According to binding type:
  :::: tabs
  
  ::: tab 0:Independent-address
    By binding a BTC deposit address to user's Vite address, one deposit address is sufficient.
  ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "depositAddress": "mrkRBVtsd96oqHLELaDtCYWzcxUr7s4D26",
        "minimumDepositAmount": "30000",
        "confirmationCount": 2,
        "noticeMsg": ""
      }
    }
  ```
  :::
  
  ::: tab 1:Bind-by-comment
    A label `memo` and corresponding value are used to mark user's Vite address when depositing EOS. User's Vite address is stored in `label`.
    
    As `memo` is used in EOS gateway, similarly, `paymentID` can be used for XMR. The 3rd party gateways can define their own `labelName`.
  ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "depositAddress": "viteeosgateway",
        "labelName": "memo",
        "label": "12345",
        "minimumDepositAmount": "30000",
        "confirmationCount": 1,
        "noticeMsg": ""
      }
    }
  ```
  :::
  ::::

:::tip Deposit Process
1. The gateway establishes the binding relationship between the **user's VITE address** and the **source chain deposit address**.
2. The gateway listens to the source chain transactions on the deposit address and waits for necessary confirmations.
3. After the gateway confirms the deposit transaction on the source chain, it initiates a transfer transaction to send the same amount of gateway tokens to user's Vite address on Vite chain.
4. The gateway listens to the transfer transaction on Vite, and must resend the transaction in case it doesn't get confirmed.
:::

### `/withdraw-info`

Get withdrawal information by token id and user's Vite address. The web wallet will display a withdrawal page based on the API response.

* **Method**: `GET`

* **Request**: `query string`

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |tokenId|Gateway token id|string|true|
  |walletAddress|User's Vite address|string|true|
  
  
* **Response**

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |minimumWithdrawAmount|Minimum withdrawal amount|string|true|
  |maximumWithdrawAmount|Maximum withdrawal amount|string|true|
  |gatewayAddress|Gateway address on Vite chain. The web wallet will send an amount of gateway tokens to the address for withdrawal|string|true|
  |labelName|Label name, required if type=1|string|false|
  |noticeMsg|Extra message filled in by gateway|string|false|

* **Example**

  ***Request***
    ```
    /withdraw-info?tokenId=tti_82b16ac306f0d98bf9ccf7e7&walletAddress=vite_52ea0d88812350817df9fb415443f865e5cf4d3fddc9931dd9
    ```
  ***Response***
    ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "minimumWithdrawAmount": "1000000",
        "maximumWithdrawAmount": "10000000000",
        "gatewayAddress": "vite_42f9a5d93e1e392624b97dfa3d7cab057b79c2489d6bc13682",
        "noticeMsg": ""
      }
    } 
    ```

### `/withdraw-address/verification`

Verify withdrawal address. The web wallet will use this API to verify the source chain withdrawal address

* **Method**: `GET`

* **Request**: `query string`

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |tokenId|Gateway token id|string|true|
  |withdrawAddress|User's withdrawal address on the source chain|string|true|
  |labelName|Label name, required if type=1|string|false|
  
  
* **Response**

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |isValidAddress|Is the user's withdrawal address valid?|bool|true|
  |message|Error message|string|false|
  

* **Example**

  ***Request***
    ```
    /withdraw-address/verification?tokenId=tti_82b16ac306f0d98bf9ccf7e7&withdrawAddress=moEGgYAg8KT9tydfNDofbukiUNjWTXaZTm
    ```
  ***Response***
    ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "isValidAddress": true
      }
    }
    ```

### `/withdraw-fee`

Get gateway withdrawal fee

* **Method**: `GET`

* **Request**: `query string`

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |tokenId|Gateway token id|string|true|
  |walletAddress|User's Vite address|string|true|
  |amount|Withdrawal amount|string|true|
  |containsFee|Is the fee included in the original withdrawal amount?<br>If this is false, then `amount` refers to actual amount transferred. The gateway will calculate the transaction fee based on this amount.<br>If this is true, then `amount` refers to the sum of actual amount transferred and the transaction fee. The gateway will subsequently derive the actual amount transferred and the transaction fee. Usually this is used in a full withdrawal.|bool|true|


  
* **Response**

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |fee|Withdrawal fee|string|true|

* **Example**

  ***Request***
    ```
    /withdraw-fee?tokenId=tti_82b16ac306f0d98bf9ccf7e7&walletAddress=vite_52ea0d88812350817df9fb415443f865e5cf4d3fddc9931dd9&amount=100000000000000&containsFee=true
    ```
  ***Response***
    ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "fee": "1000000"
      }
    }
    ```
  
:::tip Withdrawal Process
1. The user fills in a valid withdrawal amount and source chain withdrawal address, then hits the transfer button. At this time, the web wallet will send a corresponding amount of gateway tokens to the gateway address on Vite chain. The source chain withdrawal address is stored in the comment of the transaction.
2. The gateway listens for the transactions on the withdrawal address and waits for necessary confirmations.
3. After the gateway confirms the withdrawal transaction on Vite chain, it initiates a transfer transaction to send the same amount of the source chain tokens to user's withdrawal address on the source chain.
4. The gateway listens for the transfer transaction on the source chain, and must resend the transaction in case it doesn't get confirmed.
:::

#### Gateway Transaction Comment

According to [VEP 8: AccountBlock Data Content Type Definition](../../vep/vep-8.md), a transaction comment is a concatenation of fixed and variable strings. 
* Fixed part:

|VEP-8 Type|Type|
|:--|:---|
|2 Byte,uint16|1 Byte,uint8|

For gateway transactions, **VEP-8 Type** is fixed at `3011`, or `0x0bc3` in hexadecimal format

**Type** is the `type` parameter returned by `/meta-info` API

* Variable part:

  ::::: tabs
  ::: tab 0:Independent-address
  
    |Address|
    |:---:|
    |0 ~ 255 Byte,UTF-8|
    
    Taking withdrawal address `mrkRBVtsd96oqHLELaDtCYWzcxUr7s4D26` as example, the transaction comment generated in hexadecimal format is `0x0bc3006d726b52425674736439366f71484c454c6144744359577a63785572377334443236`
    
  :::
  ::: tab 1:Bind-by-comment
  
    |Address size|Address|Label size|Label|
    |:---:|:---:|:---:|:---:|
    |1 Byte,uint8|0 ~ 255 Byte,UTF-8|1 Byte,uint8|0 ~ 255 Byte,UTF-8|
    
    Taking withdrawal address `viteeosgateway` as example, having label `12345`, the transaction comment generated in hexadecimal format is `0x0bc3010d76697465746f7468656d6f6f6e053132333435`
   
  :::
  :::::

## Deposit/withdrawal Records Query API

### `/deposit-records`

Get historical deposit records

* **Method**: `GET`

* **Request**: `query string`

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |tokenId|Gateway token id|string|true|
  |walletAddress|User's Vite address|string|true|
  |pageNum|Page index, starting from 1|int|true|
  |pageSize|Page size|int|true|
  
  
* **Response**

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |totalCount|Total deposit records|int|true|
  |depositRecords|List of deposit records|list|false|
  |inTxExplorerFormat|The transaction url on the source chain explorer. Replace {$tx} with the specific `inTxHash`|string|true|
  |outTxExplorerFormat|The transaction url on Vite explorer. Replace {$tx} with the specific `outTxHash`|string|true|
  
* ***Definition of `depositRecords`***

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |inTxHash|The deposit transaction hash on the source chain|string|true|
  |inTxConfirmedCount|Confirmations conducted on source chain|int|false|
  |inTxConfirmationCount|Confirmations required on source chain|int|false|
  |outTxHash|The deposit transaction hash on Vite chain|string|false|
  |amount|Deposit amount|string|true|
  |fee|Gateway fee|string|true|
  |state|Transaction state. Allowed value: <br>`OPPOSITE_PROCESSING` Awaiting confirmation on the source chain<br>`OPPOSITE_CONFIRMED` Confirmed on the source chain<br>`OPPOSITE_CONFIRMED_FAIL` Transaction failed on the source chain<br>`BELOW_MINIMUM` Transaction aborted due to insufficient deposit amount<br>`TOT_PROCESSING` Gateway tokens sent out on Vite chain<br>`TOT_CONFIRMED` Deposit successful<br>`FAILED` Deposit failed|string|true|
  |dateTime|Deposit time in millisecond|string|true|

* **Example**

  ***Request***
    ```
    /deposit-records?tokenId=tti_82b16ac306f0d98bf9ccf7e7&walletAddress=vite_52ea0d88812350817df9fb415443f865e5cf4d3fddc9931dd9&pageNum=1&pageSize=10
    ```
  ***Response***
    ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "totalCount": 1,
        "depositRecords": [{
          "inTxHash": "0x8e791fc2430761ce82f432c6ad1614fa1ebc57b1e1e0925bd9302a9edf8fd235",
          "inTxConfirmedCount": 2,
          "inTxConfirmationCount": 12,
          "outTxHash": "9fb415eb6f30b27498a174bd868c29c9d30b9fa5bfb050d19156523ac540744b",
          "amount": "300000000000000000",
          "fee": "0",
          "state": "TOT_CONFIRMED",
          "dateTime": "1556129201000"
        }],
        "inTxExplorerFormat": "https://ropsten.etherscan.io/tx/{$tx}",
        "outTxExplorerFormat": "https://explorer.vite.org/transaction/{$tx}"
      }
    }
    ```

### `/withdraw-records`

Get historical withdrawal records

* **Method**: `GET`

* **Request**: `query string`

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |tokenId|Gateway token id|string|true|
  |walletAddress|User's Vite address|string|true|
  |pageNum|Page index, starting from 1|int|true|
  |pageSize|Page size|int|true|
  
* **Response**

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |totalCount|Total withdrawal records|int|true|
  |withdrawRecords|List of withdrawal records|list|false|
  |inTxExplorerFormat|The transaction url on Vite explorer. Replace {$tx} with the specific `inTxHash`|string|true|
  |outTxExplorerFormat|The transaction url on the source chain explorer. Replace {$tx} with the specific `outTxHash`|string|true|
  
* ***Definition of `withdrawRecords`***

  |Name|Description|Data Type|Required|
  |:--|:---|:---:|:---:|
  |inTxHash|The withdrawal transaction hash on Vite chain|string|true|
  |inTxConfirmedCount|Confirmations conducted on Vite chain|int|false|
  |inTxConfirmationCount|Confirmations required on Vite chain|int|false|
  |outTxHash|The withdrawal transaction hash on the source chain|string|false|
  |amount|Actual amount transferred|string|true|
  |fee|Gateway fee|string|true|
  |state|Transaction state. Allowed value: <br>`TOT_PROCESSING` Awaiting confirmation on Vite chain<br>`TOT_CONFIRMED` Confirmed on Vite chain<br>`TOT_EXCEED_THE_LIMIT` Transaction failed due to exceeding the maximum limit<br>`WRONG_WITHDRAW_ADDRESS` Transaction failed due to invalid withdrawal address<br>`OPPOSITE_PROCESSING` Source chain tokens sent out<br>`OPPOSITE_CONFIRMED` Withdrawal successful <br>`FAILED` Withdraw failed|string|true|
  |dateTime|Withdrawal time in millisecond|string|true|

* **Example**

  ***Request***
    ```
    /withdraw-records?tokenId=tti_82b16ac306f0d98bf9ccf7e7&walletAddress=vite_52ea0d88812350817df9fb415443f865e5cf4d3fddc9931dd9&pageNum=1&pageSize=10
    ```
  ***Response***
    ```javascript
    {
      "code": 0,
      "subCode": 0,
      "msg": null,
      "data": {
        "totalCount": 2,
        "withdrawRecords": [{
          "inTxHash": "b95c11ac34d4136f3be1daa3a9fab047e11ee9c87acef63ca21ba2cee388a80f",
          "inTxConfirmedCount": 2,
          "inTxConfirmationCount": 300,
          "outTxHash": "0x8096542d958a3ac4f247eba3551cea4aa09e1cdad5d7de79db4b55f28864b628",
          "amount": "190000000000000000",
          "fee": "10000000000000000",
          "state": "OPPOSITE_CONFIRMED",
          "dateTime": "1556129201000"
        }],
        "inTxExplorerFormat": "https://explorer.vite.org/transaction/{$tx}",
        "outTxExplorerFormat": "https://ropsten.etherscan.io/tx/{$tx}"
      }
    }
    ```
  
## Error Code
  |Code|Description|
  |:--|:---|
  |0|Request is successful|
  |1|Invalid request parameter|
  |2|Internal gateway error|

## Version of the Specification
### Current Version
`v1.0`
### Historical Version(s)
|Version|Description|
|:--|:---|
|v1.0|Initial version|

