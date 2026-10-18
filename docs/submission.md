# Prototype Submission

## Mandatory Submission Rules

1. **Pitch Deck**
   Participants must submit their pitch deck using the prescribed template provided for the challenge. Submissions that do not follow the required format may be subject to disqualification.

2. **Video Demonstration**
   Participants must submit a 3–4 minute video demonstration of their solution. The video must clearly demonstrate the working functionality of the submitted solution.
   - Videos exceeding the prescribed duration may attract a penalty or result in the submission not progressing further in the evaluation process.
   - If the video does not adequately demonstrate the working functionality of the solution, the submission will not progress to the prototype evaluation step, in accordance with the defined ladder process.

3. **Deployed Prototype Link**
   Participants must provide a valid and accessible link to a deployed, working prototype. The prototype must remain functional throughout the evaluation process.
   - If the application, APIs, integrations, or any other deployment-related components are unavailable or non-functional during evaluation, no additional consideration, retries, or allowances will be provided.
   - The submission will be evaluated based on its functionality at the time of evaluation.

4. **GitHub Repository**
   Participants must provide an accessible GitHub repository containing the relevant source code and supporting materials. The repository must remain accessible throughout the evaluation process.

5. **Brief Description**
   A brief description of what you have built (please include how you are leveraging Firebase, Firestore, Cloud Run, and Gemini in your submission).

> **Instructions:** Make sure all the submission links are working and publicly accessible, and include all the tech mentioned in the submission rules.

## Challenge Details

|                   |                                          |
| ----------------- | ---------------------------------------- |
| **Challenge**     | Future of Work & Enterprise Productivity |
| **Starting Date** | 07/09/2026 05:30 PM (KST)                |
| **Ending Date**   | 19/10/2026 03:29 AM (KST)                |

## Deliverables

- [ ] **Prototype Link** — Deployed on GCP / Cloud Run

      https://clustermind-607541401069.asia-northeast3.run.app/

- [ ] **Final Prototype PPT/Deck** — Converted into PDF (chkwak, to be updated)

      ClusterMind_AI_Builder_Cup.pdf

- [x] **GitHub Repository Link** — Public access (completed)

      https://github.com/AIBuilderCup-ClusterMind/ClusterMind.git

- [ ] **Demo Video Link** — 3–4 minutes (chkwak, to be updated)

      https://www.youtube.com/watch?v=ABtdFjtSfXg

- [ ] **Brief Description of Your Solution** — Include how you are leveraging Google services in your submission (chkwak, to be updated)

      **ClusterMind** is an AI-powered Kubernetes digital twin that turns a live cluster into an interactive topology map. It collects cluster resources strictly read-only, infers a dependency graph from label selectors, ingress backends, HPA targets, and config references (with per-edge confidence scores), and renders system health on an interactive React Flow canvas served by a Go/Gin REST API and a React/TypeScript SPA. On top of the graph it derives sev1–sev3 incident timelines from Kubernetes events, computes failure blast radius with a five-hop BFS traversal, and produces AI investigation reports that link causes to evidence and suggest kubectl next checks.

      Live links: app https://clustermind-607541401069.asia-northeast3.run.app · intro page https://clustermind-intro-607541401069.asia-northeast3.run.app

      **How we leverage Google Cloud technologies:**

      - **Gemini (`google.golang.org/genai` SDK, `gemini-3.5-flash-lite`)** — The core AI engine. Gemini powers (1) incident root-cause investigation: graph topology and event-timeline context are JSON-serialized into structured prompts that request structured JSON reports (summary, observed facts, ranked causes with confidence, evidence links, next checks), and (2) failure impact analysis, where a deterministic five-hop BFS blast radius is enriched with business-impact reasoning and severity scoring. When no API key is configured or the model is unreachable, ClusterMind falls back to a deterministic report so the demo never fails.
      - **Cloud Run** — The entire application ships as a single distroless, non-root container built by Cloud Build: a multi-stage Dockerfile compiles the React frontend and embeds it into the Go binary (`embed.FS`), so one image serves both the SPA and the REST API with dynamic `PORT` binding and scale-to-zero (`--min-instances=0`). The demo app, the incident-investigation flow, and the bilingual intro page all run on Cloud Run in `asia-northeast3`. A clusterless demo mode (`CLUSTERMIND_DEMO=failure`) lets the deployed prototype demonstrate a full failure scenario — broken `payment-deployment`, derived sev1 incident timeline, and Gemini investigation — without any backing cluster, which is what the public demo link runs today.
      - **Secret Manager** — `GEMINI_API_KEY` is stored in Secret Manager and injected into the Cloud Run revision as a runtime secret; for real-cluster deployments the generated container kubeconfig (short-lived ServiceAccount token) is stored the same way and mounted as a volume. No credentials are ever baked into the image or the repository.
      - **Artifact Registry + Cloud Build** — Both deploy scripts use `gcloud run deploy --source`, which builds the container image via Cloud Build and stores it in Artifact Registry (APIs and IAM bindings provisioned idempotently by the scripts).
      - **GKE (remote-cluster mode)** — `cloudrundeploy.sh` connects Cloud Run to an explicitly targeted remote GKE cluster using least-privilege access: a namespace-scoped reader Role plus `roles/container.viewer` for the Cloud Run service identity, and a TokenRequest-generated short-lived SA token — no cluster-admin, no static secret tokens. ClusterMind never writes to the cluster and deliberately never collects Kubernetes Secrets.
