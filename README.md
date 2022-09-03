# FNRadio

FNRadio changes the music played on Fortnite's [radio stations](https://fortnite.fandom.com/wiki/Radio_Stations) using a MiTM web proxy.

**Discord:** https://discord.gg/bgRM3XdhnA

# Station types
`static` - This is how Fortnite's radio works. You'll want to have a long collection of songs, and fortnite will start playing with a random start position.
`stream` - This is similar to a discord music bot. You can create a station where you can queue songs realtime with the play command.

# How to use

**Make sure to start FNRadio before launching fortnite.**

`create nurture static https://www.youtube.com/watch?v=iuBdQf345Qo` - creates a new station with the id `nurture` and audio of that youtube link

`create nurture static https://www.youtube.com/playlist?list=PLfiMjLyNWxeZdzg5XuoPAggaPJu_TYf_6` - creates a new station with the id `nurture` containing every song from that playlist

`create example stream` - creations a new stream station with the id `example`

`play example https://www.youtube.com/watch?v=dQw4w9WgXcQ` - queues that song on the example station (if the station specified is a static station, it will replace the track)

`bind example Icon Radio` - this makes the contents of your example station play on the Icon Radio in-game station

`unbind Icon Radio` - this makes the Icon Radio station revert to the normal audio
