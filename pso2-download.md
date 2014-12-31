## pso2-download

[**Download pso2-download.exe**](pso2-download.exe) - [Screenshot!](pso2-download-screen.png)

pso2-download is an updater, launcher, and language patcher for Phantasy Star Online 2. It automatically performs quick updates and keeps patches instantly up to date as soon as the game is updated. Patches are downloaded in a very bandwidth-conscious way such that the entire game can be translated with about 6MB.

Set it up once, forget about it, and just use it to launch pso2 from now on. The game and any patches will automatically be kept up to date for you. To start using it, do the following...

1. Create a shortcut to the exe (hold alt and drag pso2-download.exe to your desktop, for example)

2. Open up the shortcut's properties

3. In the Shortcut tab, add any flags you want to use to the end of the Target line. The following flags are recommended for a full english patch:

        -l -d -i -t eng,story-eng -b

    You may omit `-i` to disable the item translation, and the `-t eng,story-eng` part to disable the english patch. Get rid of `-l` if you only want to use the update and patching functionality.
4. After all flags, add the path to your pso2_bin folder. Like this example Target:

        "C:\pso2-download.exe" -l -d -i -t eng,story-eng -b -pubkey publickey.blob "C:\Program Files (x86)\SEGA\PHANTASYSTARONLINE2\pso2_bin"

5. In the Compatibility tab, check "Run this program as an administrator" (optional, but required if you want it to actually launch the game. Skip if you only use this as a downloader/patcher).

6. Launch the shortcut. It will do any downloading/updating/patching only when necessary, then start the game for you.


**Warning** upon first run the launcher will need to do a quick file check. If it finds anything modified (such as english patches), the files will be redownloaded and then repatched. You may want to uninstall your patches from a backup beforehand to avoid any wasted bandwidth.


### Other Usage Examples

- Revert your pso2 installation to an original, unpatched state by hash checking all files and downloading any that have been modified/corrupt/etc.

        "C:\pso2-download.exe" -h -a -d "C:\Program Files (x86)\SEGA\PHANTASYSTARONLINE2\pso2_bin"

    Note, you may want to copy the files from `pso2_bin/download/backup` beforehand (if it exists) in order to lessen the load of files that need redownloading.

- Remove any files wasting space in your PSO2 folder

        "C:\pso2-download.exe" -u -g "C:\Program Files (x86)\SEGA\PHANTASYSTARONLINE2\pso2_bin"


### Flags

There are a whole lot of flags that pso2-download.exe accepts.

- `-l` Launch the game after performing all other operations
- `-d` Download any files that have changed since the last update
- `-i` Use the english item translation patch
- `-t` Apply the specified english translations by name (currently only eng and story-eng exist)
- `-b` Back up any files before patching them. This places files under `pso2_bin/download/backup`, which can be copied back into `data/win32` to restore the game without a huge download
- `-pubkey path.blob` Inject the specified public key into PSO2. Used for [PSO2Proxy](http://pso2proxy.cyberkitsune.net)
- `-a` Consider all files unupdated. Useful in conjunction with -c and -h
- `-c` Check files to determine whether any have been changed
- `-h` Hash all files when checking them. A more time-intensive but thorough version of the -c flag
- `-g` Clean up any unused garbage files left over from previous updates/installs
- `-u` Force a redownload of the file list without checking for a new version
- `-dumppubkey path.blob` dumps the Sega public key to a file
