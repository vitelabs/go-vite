---
order: 4
---

# Installation Guide for Ledger Nano S

The **Ledger Vite App** has completed development and has been submitted to Ledger for formal review. Since the review is still in progress, users who want to use Ledger hardware wallet to manage assets on Vite are unable to install the app directly through Ledger Live at the moment. 
This guide explains how to install **Ledger Vite App** on Ledger Nano S from source code.
1. Install VirtualBox;
2. Install Ubuntu on VirtualBox;
3. Download the source code and configure the compilation environment;
4. Compile and install.

Please strictly follow the below steps.

## Install VirtualBox
Download VirtualBox and Extension Pack at [VirtualBox Download Page](https://www.virtualbox.org/wiki/Downloads). Follow the instructions to install VirtualBox first, and then install the Extension Pack.

![download-virtualbox](./assets/ledger-install/download-virtualbox.png)

## Install Ubuntu on VirtualBox
Download the Ubuntu installation file at [Ubuntu Download Page](https://ubuntu.com/#download). Click the button shown in the figure below to start download.

![download-ubuntu](./assets/ledger-install/download-ubuntu.png)

Open VirtualBox and click New to create a new virtual machine.

![install-1](./assets/ledger-install/install-1.png)

Fill in a name and select a suitable location to store the VM files. In this guide, we use "ubuntu" and the default location.
Note that the Type must be **Linux**, and the Version must be **Ubuntu (64-bit)** to ensure that it is consistent with the installation image we downloaded earlier. Then click Continue.

![install-2](./assets/ledger-install/install-2.png)

Set the memory size to 2048 MB or higher, click Continue.

![install-3](./assets/ledger-install/install-3.png)

Select "Create a virtual hard disk now", click Create.

![install-4](./assets/ledger-install/install-4.png)

Select VDI (VirtualBox Disk Image), click Continue.

![install-5](./assets/ledger-install/install-5.png)

Select "Dynamically allocated", click Continue.

![install-6](./assets/ledger-install/install-6.png)

Select the location where the new hard disk will be stored. Here we use the default location and the recommended hard disk size, click Create.

![install-7](./assets/ledger-install/install-7.png)

Now the VM is successfully created, load the Ubuntu installation image and click Settings

![install-8](./assets/ledger-install/install-8.png)

Click Storage, select Empty under "Controller: IDE", and click the CD icon pointed by the arrow on the right.

![install-9](./assets/ledger-install/install-9.png)

Select Choose/Create a Virtual Optical Disk... 

![install-10](./assets/ledger-install/install-10.png)

Click Add to add the image file.

![install-11](./assets/ledger-install/install-11.png)

Select the previously downloaded ubuntu-20.04-desktop-amd64.iso image file, click to open.

![install-12](./assets/ledger-install/install-12.png)

Select the image file and click Choose.

![install-13](./assets/ledger-install/install-13.png)

As shown in the figure below, the image file has been added, click OK to save the settings.

![install-14](./assets/ledger-install/install-14.png)

Click Start to start the virtual machine.

![install-15](./assets/ledger-install/install-15.png)

Now finish the Ubuntu installation with the below steps.

Click Install Ubuntu.

![install-16](./assets/ledger-install/install-16.png)

Click Continue.

![install-17](./assets/ledger-install/install-17.png)

Select Minimal installation, uncheck Download updates while installing Ubuntu, and then click Continue.

![install-18](./assets/ledger-install/install-18.png)

Select "Erase disk and install Ubuntu", then click Install Now.

![install-19](./assets/ledger-install/install-19.png)

Click Continue.

![install-20](./assets/ledger-install/install-20.png)

Select your region, then click Continue.

![install-21](./assets/ledger-install/install-21.png)

Enter the username and password, select "Log in automatically", and then click Continue.

![install-22](./assets/ledger-install/install-22.png)

Now the Ubuntu installation will start. Please wait in patience. After the installation is complete, click Restart Now to restart the system.

![install-23](./assets/ledger-install/install-23.png)

## Prepare the source code compilation environment
Click Activities, input "terminal" in the search box, and click to open.

![](./assets/ledger-install/config-1.jpg)

![](./assets/ledger-install/config-2.jpg)

Execute the following command to install the required software packages.
```shell
sudo apt install git build-essential autoconf python3-venv python3-dev libudev-dev libusb-1.0-0-dev gcc-arm-none-eabi gcc-multilib g++-multilib libtinfo5
```
During the installation process, you will be asked to enter the root password. When asked whether to continue, type "Y".

After the installation is successful, execute the following command to download the source code and prepare the compilation environment.
```shell
git clone https://github.com/vitelabs/ledger-app-vite.git
cd ledger-app-vite/
source prepare-devenv.sh
```

## Compile and Install
Insert the Ledger Nano S device into the USB port of the computer.

![ledger-1](./assets/ledger-install/ledger-1.jpg)

Then enter the PIN code on the device.

![ledger-2](./assets/ledger-install/ledger-2.jpg)

Click the USB icon in the lower right corner of the VM window, select Ledger Nano S to make sure the device is connected to the virtual machine.

![build-1](./assets/ledger-install/build-1.png)

![build-2](./assets/ledger-install/build-2.png)

Run the following command to compile and install the Vite App firmware to the Ledger Nano S device.
```shell
make load
```
During the installation process, you will be asked to enter the PIN code, just follow the instructions.

![ledger-3](./assets/ledger-install/ledger-3.jpg)

![ledger-4](./assets/ledger-install/ledger-4.jpg)

![ledger-5](./assets/ledger-install/ledger-5.jpg)

![ledger-6](./assets/ledger-install/ledger-6.jpg)

![ledger-7](./assets/ledger-install/ledger-7.jpg)

Now Vite should appear on the device's main screen. Congratulations, you have successfully installed the Vite App on the device. 

:::tip Uninstall
Execute the following command and follow the instructions if you need to uninstall Vite App from the device.
```shell
make delete
```
:::
