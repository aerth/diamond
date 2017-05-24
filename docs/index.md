# diamond

### add runlevels to your application

Some reasons to use `diamond` include:

  * Remove downtime while redeploying
  * Have more control of your application (not just start and stop)
  * Quickly disable all tcp/socket listeners
  * ... and quickly re-enable existing or additional listeners
  * Have two applications sharing a single port or http socket (one at a time)
  * Use shell or SSH to trigger events in your application
  * Schedule listeners via crontab, systemd unit scripts, or in your Go code
  * Testing, making the `diamond` library more stable and useful

Some cool features include:

  * Control socket (kind of [but not] like tmux)
  * Command line client for connecting to control socket
  * Close, Reopen 'TCP' or 'unix' listeners
  * The 'Kick' feature

About KICK:

  * KICK is sort of like sending SIGHUP to the program, but it is via the control socket
  * When booting up, if configured as KICKS, if the `control socket` exists, it will be send a KICK command
  * If the diamond system is configured to be KICKABLE, it will respond with OKAY and run Runlevel(0)
  * If the response is OKAY, the new booting diamond will then create the socket and begin
  * If the response is NOWAY, the new booting diamond will exit with an error
  * The OKAY response blocks until the socket is made and accepts connections
  * If a `diamond client` sends a KICK command, it is the same as the `runlevel 0` command
  * For expected results, the running `diamond server` must be configured as `Kickable` and the one booting configured as Kicks


