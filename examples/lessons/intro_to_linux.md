# Introduction to Linux Commands

This lesson introduces basic Linux commands for navigating the file system and working with files.

## Step 1: Checking Your Current Location

Let's start by exploring the file system. The `pwd` command (Print Working Directory) shows your current location in the file system:

```docker
pwd
```

```expect
/home/user
```

## Step 2: Listing Files and Directories

Now let's list the files and directories in your current location. The `ls` command (List) shows the contents of a directory:

```docker
ls -la
```

The `-l` option shows detailed information, and the `-a` option shows hidden files (those starting with a dot).

## Step 3: Creating Directories

Let's create a new directory for our work. The `mkdir` command (Make Directory) creates a new directory:

```docker
mkdir my_project
ls -l
```

```expect
total 4
drwxr-xr-x 2 user user 4096 Jan 1 00:00 my_project
```

## Step 4: Changing Directories

Now let's move into our new directory. The `cd` command (Change Directory) changes your current location:

```docker
cd my_project
pwd
```

```expect
/home/user/my_project
```

## Step 5: Creating Files

Let's create a new file in our directory. The `echo` command can be used with redirection (`>`) to create a file:

```docker
echo "Hello, Linux World!" > hello.txt
ls -l
```

```expect
total 4
-rw-r--r-- 1 user user 19 Jan 1 00:00 hello.txt
```

## Step 6: Viewing File Contents

Now let's look at the contents of our file. The `cat` command (Concatenate) displays the contents of a file:

```docker
cat hello.txt
```

```expect
Hello, Linux World!
```

## Step 7: Copying Files

Let's make a copy of our file. The `cp` command (Copy) creates a copy of a file:

```docker
cp hello.txt hello_backup.txt
ls -l
```

```expect
total 8
-rw-r--r-- 1 user user 19 Jan 1 00:00 hello_backup.txt
-rw-r--r-- 1 user user 19 Jan 1 00:00 hello.txt
```

## Step 8: Moving and Renaming Files

The `mv` command (Move) can be used to move or rename files:

```docker
mv hello.txt greeting.txt
ls -l
```

```expect
total 8
-rw-r--r-- 1 user user 19 Jan 1 00:00 greeting.txt
-rw-r--r-- 1 user user 19 Jan 1 00:00 hello_backup.txt
```

## Step 9: Removing Files and Directories

Finally, let's clean up. The `rm` command (Remove) deletes files, and `rmdir` removes empty directories:

```docker
rm greeting.txt hello_backup.txt
ls -l
cd ..
rmdir my_project
ls -l
```

```question
What command would you use to see the manual page for a command like `ls`?
```

## Step 10: Getting Help

Most Linux commands have built-in help. You can use the `--help` option or the `man` command to get more information:

```docker
ls --help | head -n 5
man ls | head -n 5
```

Congratulations! You've learned the basic Linux commands for navigating the file system and working with files.