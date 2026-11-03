export function BackendUnavailableBanner() {
  return (
    <div className="banner banner-error" role="alert">
      Backend disconnected or unreachable. Start AI Hub on port 8080 and use Reconnect SSE after it is healthy.
    </div>
  )
}
