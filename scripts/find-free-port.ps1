param(
    [Parameter(Mandatory = $true)]
    [int] $StartPort,

    [Parameter(Mandatory = $true)]
    [int] $EndPort
)

for ($port = $StartPort; $port -le $EndPort; $port++) {
    $listener = $null
    try {
        $listener = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Any, $port)
        $listener.Start()
        Write-Output $port
        exit 0
    } catch {
        # Port is already in use or blocked by the OS; try the next one.
    } finally {
        if ($null -ne $listener) {
            $listener.Stop()
        }
    }
}

exit 1
