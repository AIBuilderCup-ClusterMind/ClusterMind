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

- [ ] **Prototype Link** — Deployed on GCP / Cloud Run (Arpita Pani, Checking)
- [ ] **Final Prototype PPT/Deck** — Converted into PDF (chkwak, to be updated)

      ClusterMind_AI_Builder_Cup_Submission_draft.pptx

- [x] **GitHub Repository Link** — Public access (completed)

      https://github.com/AIBuilderCup-ClusterMind/ClusterMind.git

- [ ] **Demo Video Link** — 3–4 minutes (chkwak, to be updated)

      https://drive.google.com/file/d/11syWDjm-r1Q426zWms1LRN8cgI9PqOOw/view

- [ ] **Brief Description of Your Solution** — Include how you are leveraging Google services in your submission (chkwak, to be updated)

      **ClusterMind** is an AI-powered Kubernetes digital twin that transforms live cluster infrastructure into an interactive topology map, calculates failure blast radius, and delivers incident investigations. Built for enterprise reliability, it collects cluster resources strictly read-only, infers dependency edges with confidence scores, and visualizes system health via an interactive React Flow canvas backed by a Go/Gin REST API.

      **How We Leverage Google Cloud Technologies:**
      - **Gemini (`google.golang.org/genai` SDK):** Powers AI incident investigations and blast-radius impact analysis. ClusterMind constructs context-rich graph and timeline prompts, requests structured JSON reports with root causes and remediation steps, and falls back to deterministic analysis when offline.
      - **Cloud Run & GKE:** The entire application is packaged as a distroless container (embedding both the compiled React frontend and Go backend) and deployed on Cloud Run with dynamic port binding. It connects to an explicitly specified remote GKE target restricted to a dedicated demo namespace via least-privilege RBAC (no Secrets collection) and short-lived TokenRequest / service identity authentication. Secret Manager securely injects the container kubeconfig and `GEMINI_API_KEY`.
