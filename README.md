# Arris XMLC Configuration Decrypt/Encrypt Tool

Arris home internet routers allow configuration files to be saved and loaded for the purposes of backing up and restoring the router configuration. The configuration is saved as an encrypted XML file with an `XMLC` header, requiring programmatic decryption before the contents can be viewed in plain text. To change the configuration data, an updated XML file must then be encrypted using the expected algorithm.

This XMLC tool can perform both functions: decrypting a saved configuration file into XML, and encrypting XML into a file that can be loaded. This repository contains 2 implementations of the same tool, with the original written in Python and the newer implementation written in Go.

This utility was created for and tested using an Arris NVG468MQ router using firmware version 9.3.0.

## Usage

### Always Save a Backup

**IMPORTANT**: Before modifying the configuration of your router, be sure to save an unmodified copy of the `config.dat` file for restoration in case of a modification causing corruption.

It can also be helpful to restore the router to factory defaults and save a copy of that configuration.

### Python

The Python implementation requires the `pycryptodomex` library, with version `3.19.0` or greater. This can be installed using `pip install pycryptodomex`.

1. Use the web interface for "Save Configuration" to retrieve the encrypted `config.dat` file
2. Run `python3 xmlc.py config.dat config.xml` to decrypt `config.dat` into `config.xml`
3. Make changes to `config.xml` as desired
4. Run `python3 xmlc.py -c config.xml config.dat` to encrypt `config.xml` back into `config.dat`
5. Use the web interface for "Load Configuration" to load the modified configuration

### Go

The Go implementation does not require any prerequisite installations.

1. Use the web interface for "Save Configuration" to retrieve the encrypted `config.dat` file
2. Run `go run xmlc.go config.dat config.xml` to decrypt `config.dat` into `config.xml`
3. Make changes to `config.xml` as desired
4. Run `go run xmlc.py -c config.xml config.dat` to encrypt `config.xml` back into `config.dat`
5. Use the web interface for "Load Configuration" to load the modified configuration

## Configuration Editing

The Arris router offers a wide array of configuration options, but can suffer from some bugs and limitations in the web interface. Directly editing the XML configuration allows those issues to be bypassed (while incurring risk of corrupting the configuration due to mistakes). "Firewall - Access Control" editing is a prime example where the web interface cannot be used to configure the settings exactly as desired, but editing the XML by hand does.

### Firewall - Access Control

Per the help displayed on this page, "Each profile can specify one or two access time ranges for each day of the week." However, the web interface suffers from a bug where two access time ranges cannot be configured such that access is allowed over the midnight boundary. Consider the desired configuration of allowing access until 1:00 AM each night, resuming at 7:00 AM the next morning. Because the configuration is based on the day of the week, this would require two ranges of allowed access:

1. Midnight to 01:00 AM
2. 07:00 AM to 11:59 PM

When attempting to create this configuration though, the web interface prevents the second range from being added (regardless of order), showing an error message:

> (0) settings incompatible

If you choose "11:30 PM" instead of "11:59 PM", the changes will be saved as expected, but this leaves a window of 30 minutes where access will be blocked. This limitation seems to be a bug in how the web interface validates the compatibility of the two ranges configured for any given day, but this can be worked around by using the XMLC tool and hand-editing the configuration.

To set up a configuration with such access ranges:

1. Navigate to Firewall > Access Control
    1. Create a new Access Control Profile, giving it your desired name
    2. Click to "Edit" the new profile, which will show access is allowed all day by default
    3. Using "Every day", choose the following values and click "Add to Profile":
        a. Access Begins at: Midnight
        b. Access Ends at: Desired end time (for the nighttime shut-down time after midnight)
    4. Using "Every day", choose the following values and click "Add to Profile":
        a. Access Begins at: Desired start time (for the morning access beginning time)
        b. Access Ends at: 11:30 PM
2. Navigate to Advanced > Configuration
    1. Save Configuration File
3. Run the XMLC tool to decrypt `config.dat` into `config.xml`
4. Edit the `config.xml` file using a text editor
    1. Find the Access Control Profile data, which will be within a `<profile name="{name}" id="{id}">` element where your profile name appears
    2. For each day, change the `<end2>` element value from `84600` to `86399`
        - These values represent the number of seconds since midnight
        - A value of `84600` represents 11:30:00 PM
        - A value of `86399` represents 11:59:59 PM, which is the max value for any given day
    3. For each day, change the `<max-usage>` element value to add `1799` seconds to the value
        - This will add the 29 minutes and 59 seconds of max usage back to the day
        - After adding that amount, the max usage total will match the combined lengths of the access ranges
    4. Save the updated XML file
5. Run the XMLC tool to encrypt `config.xml` back to a `.dat` file using a new file name
6. Navigate to Advanced > Configuration
    1. Choose your modified `.dat` file for "Load Configuration File"
    2. Click "Load"
    3. Patiently wait while the router loads the configuration and restarts
7. Navigate to Firewall > Access Control
    1. Click to "Edit" the profile that you created and edited the XML configuration for
    2. Verify that the access time ranges match your expectation
    3. If the "Access Begins at" or "Access Ends at" time shows as `nil` in the web interface, it's likely that the values is incorrect or that the `<max-usage>` value does not match the total of the two range durations and the math needs to be corrected
