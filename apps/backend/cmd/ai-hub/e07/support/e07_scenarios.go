package support

func E07ReplayFixture() (E07CodexRun, E07CodexRun) {
	first := E07CodexRun{Ran: true, ExitCode: 0, ThreadID: "thr_synthetic_identifier"}
	second := E07CodexRun{Ran: true, ExitCode: 0, Message: "the marker is " + E07ReplayMarker}
	return first, second
}
