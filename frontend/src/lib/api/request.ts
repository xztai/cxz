// Called for every fetch, including the CSRF fetch. A shared timeout signal would
// expire once and permanently break the stateless client after that point.
export function requestWithTimeout(request: Request): Promise<Response> {
	return fetch(request, {
		signal: AbortSignal.any([request.signal, AbortSignal.timeout(10_000)])
	});
}
