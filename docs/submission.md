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

      **ClusterMind** is an AI-powered digital twin for Kubernetes that turns a live cluster into an interactive topology map and predicts failure blast radius. It collects resources strictly read-only, infers dependency edges (Label Selectors, Ingress, HPA, ConfigMap mounts) with confidence scores, and renders them as a draggable React Flow canvas via a Go/Gin REST API.

      Gemini (via the official google.golang.org/genai SDK) is our core reasoning engine: it explains architecture, reasons about business impact on top of BFS graph traversal, assigns severity, and generates incident investigation reports from Kubernetes Events, with a deterministic fallback. We deploy on Cloud Run — Cloud Build builds one container image (React SPA + API) stored in Artifact Registry, and Secret Manager holds the read-only kubeconfig and GEMINI_API_KEY with least-privilege runtime mounting. The stateless prototype stays live throughout evaluation; source: https://github.com/AIBuilderCup-ClusterMind/ClusterMind
