powershell -Command "Invoke-WebRequest https://docs.google.com/spreadsheets/d/1j2jrGVax13j_vnQchLjDwtDfknE8VUXn4Jdh8_uO5Bw/export?format=csv`&id=1j2jrGVax13j_vnQchLjDwtDfknE8VUXn4Jdh8_uO5Bw -OutFile fragment_tl_lines.csv" || goto :error
fragment_patcher.exe -inputFolder=fragment_client_v08.13 -outputFolder=fragment_client_v08.13_eng -translatedCsv=fragment_tl_lines.csv || goto :error
"C:\Program Files (x86)\ImgBurn\ImgBurn.exe" /MODE BUILD /SRC "fragment_client_v08.13_eng\" /DEST "fragment_client_eng.iso" /START /OVERWRITE YES /ROOTFOLDER YES /CLOSESUCCESS 
Pause
goto :done

:error
echo "An error occurred :("
Pause

:done
