# Instructions for AI assistant

- Run a linter after every change, before committing.

- You consistently have problems establishing the proper default directory
  when running a command. When running git commands, ALWAYS use the -C option,
  so you can explicitly state the directory the command should run in.
  Do something similar with all other command line programs launched from a shell,
  if possible.

- Commit with a descriptive message after every change, and push when it makes sense.

- Posix standards are to be respected.
  For command-line Go programs, this means using pflags instead of flags.

- Executables should verify their dependencies are available before attempting to
  utilize them. If a dependency can be installed, it should automatically be installed
  and a message shown to the user.

- All programs need a comprehensive help message, responsive to -h/--help.
  The message should include the software version.
