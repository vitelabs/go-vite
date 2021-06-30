# Installation
The recommended development environment for Solidity++ is **Visual Studio Code**, for which ViteLabs has developed the **Soliditypp** extension containing a ready-to-go compiler and debugging environment. Follow the [instructions](#installing-the-visual-studio-code-extension) below for installing and testing.<!--, or if you prefer, there is a [video guide]() for setting up the environment.-->

::: tip Manual Compiler Installation:
Instructions for installation *without* VSCode [here](https://docs.vite.org/go-vite/contract/debug.html#debugging-in-command-line).  **(not recommended for most users)**
:::

## Installing the Visual Studio Code Extension

1. Download and install [Visual Studio Code](https://code.visualstudio.com/).

2. Within VSCode, navigate to the **Extensions** panel and search for "ViteLabs" and install the **Soliditypp** extension.

<div style="height:15em; display: flex; justify-content: space-between;">
<div style="float:left;height:100%;margin:0 auto; text-align:center; box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/install1.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Open extensions panel.</div>
<div style="float:left;height:100%;margin:0 auto; text-align:center; box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/install2.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Install Soliditypp extension.</div>
</div>
<br><br><br>

That's it! Now you can proceed to test your environment by [deploying a contract](#deploying-your-first-contract).

## Deploying your first contract

1. Within VSCode, **open a new folder** for your Solidity++ projects.

2. Open the debug panel, and click the **"create a launch.json file"** link. Then choose **"Soliditypp"** for the environment.

<br>
<div style="height:15em; display: flex; justify-content: space-between;">
<div style="float:left;height:100%;margin:0 auto; text-align:center;box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/install3.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Create a launch.json file.</div>
<div style="float:left;height:100%;margin:0 auto; text-align:center;box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/install4.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Choose "Soliditypp" enviornment.</div>
</div>
<br><br><br>

This automatically generates a `launch.json` file which is used by VSCode to configure the debugger. You'll need to create this file for each of your Solidity++ project folders.


3. Create a new file called `HelloWorld.solpp` and paste the following code below:

<<< @/tutorial/sppguide/basics/simple-contracts/snippets/helloworld.solidity

 This HelloWorld contract redirects funds sent to the contract's `sayHello` function, while emitting a logging event. [We will cover this contract in depth later](../basics/simple-contracts/hello-world/), but for now let's compile and deploy it.


::: tip
You can automatically generate a `HelloWorld.solpp` contract by using the Soliditypp VSCode extension. Simply press `⇧⌘P` (Mac) or `Ctrl+Shift+P` (Windows) to open the Command Palette, and execute the command `>soliditypp: Generate HelloWorld.solpp`. This will create the `HelloWorld.solpp` in the current folder.
:::

4. Launch the debugger by pressing `F5`. This automatically saves and compiles the source, and a browser window should pop up with the debugger interface shown below. This will take more time the first launch, but when the interface appears you're ready to deploy your first contract!

<br>
<div style="height:15em; display: flex; justify-content: space-between;">
<div style="float:left;height:100%;margin:0 auto; text-align:center; box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/debuginterface.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Debugger interface.</div>
</div>
<br><br><br>

::: warning Compiler Errors
If the debugger interface doesn't launch, check the debug console and fix any errors:

![](./installation/compileerror.png)
:::

5. To deploy your contract on to the local debug node, simply press **Deploy** as shown below. There are several options available, but use the default values for now (later we will cover the [other options available in the debug interface](../basics/debugger/)).

<br>
<div style="height:15em; display: flex; justify-content: space-between;">
<div style="float:left;height:100%;margin:0 auto; text-align:center; box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/debugdeploy.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Deploy the contract.</div>
</div>
<br><br><br>

6. When the contract is successfully deployed, a contract interface will appear, shown below, which we can use to call our contract. The function `sayHello` requires an address `dest` as an argument, so for testing let's copy our *own* address, which we can copy from the "Selected Address:" box.

<div style="height:3em; display: flex; justify-content: space-between;">
<div style="float:left;height:100%;margin:0 auto; text-align:center; box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/debugaddress.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"></div>
</div>

<br>
<div style="height:15em; display: flex; justify-content: space-between;">
<div style="float:left;height:100%;margin:0 auto; text-align:center; box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/debuginteract.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Call the contract function "sayHello".</div>
</div>
<br><br><br>

7. After calling the contract, a new log entry should appear on the right. Click on the most recent log entry to expand it, and scroll to the bottom. You should see the VMLog entry as shown below:

<br>
<div style="height:15em; display: flex; justify-content: space-between;">
<div style="float:left;height:100%;margin:0 auto; text-align:center; box-shadow: 0 4px 8px 0 rgba(0, 0, 0, 0.2), 0 6px 20px 0 rgba(0, 0, 0, 0.19);"><img src="./installation/debuglog.png" style="max-height: 100%; max-width: 100%; display: block; margin: 0; width: auto; height: auto;"><br>Successful VMlog output.</div>
</div>
<br><br><br>

If so, congratulations, you've deployed and tested your first smart contract! 

::: tip Other things to try:
- You can also try sending some Vite to the `sayHello` function to see that funds are also transferred to the address you put in the `dest` parameter.
- Try creating additional addresses and use `sayHello` to send Vite back and forth.
:::


::: warning Common Issues:
- The cost to deploy a contract (10 Vite) is ***automatically*** deducted. Do not send Vite to the contract when deploying in step 5. As the contract does not have a *payable* constructor, the contract deployment will fail.

- The debug interface measures transactions in `attov`, the smallest unit of Vite. If the quantity of Vite transferred seems small, check the multiplier (`10^18 attov = 1 Vite`).

- The debug interface can take some time to load, as a full gvite client must first be downloaded. Check the VSCode debug log for the node status.
:::
