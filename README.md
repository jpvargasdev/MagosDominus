# ⚙️ Magos_Dominus

“The Machine does not err. The flesh errs. The Code is truth, and I am its voice.” – Credus del Adeptus Mechanicus

## 📜 About

Magos_Dominus is a lightweight GitOps agent forged in the spirit of the Adeptus Mechanicus. Its sole purpose: to enforce the declared state from your sacred repository (Git) and reconcile it with the material reality of your homelab.

Unlike the bloated rites of Kubernetes and endless CRDs, Magos_Dominus imposes order directly on a simple Linux server with Podman Compose. No unnecessary ceremony, no wasted bureaucracy – only obedience to the written mandate.

Each reconciliation loop is a ritual. Each deployment, a litany. Where drift appears, corruption is purged. Where the manifest and the machine diverge, the Magos enforces the will of the Code.

⸻

## ✨ Features (planned)
	•	Registry watcher (GHCR first, later DockerHub/Quay).
	•	ImagePolicy evaluation:
	•	Semantic version ranges (>=1.2.0 <2.0.0)
	•	Regex filters on tags
	•	Architecture constraints (e.g. amd64)
	•	Min age delays to avoid race conditions
	•	(Optional) Signature verification with cosign
	•	(Optional) Vulnerability scanning (Trivy)
	•	GitOps reconciler:
	•	Patches Compose files with immutable digests (@sha256:...)
	•	Commits and pushes via GitHub App (no long-lived tokens)
	•	Direct push to main or PR workflow
	•	Optional applier:
	•	Decrypts secrets with SOPS
	•	Runs podman-compose pull && up -d for the affected stack
	•	State & observability:
	•	Minimal local cache (last digest applied)
	•	Structured logs
	•	/healthz and /metrics endpoint (future)
  
## 📂 Repository layout

cmd/mini-flux/        # entrypoint
internal/watcher/     # registry polling & events
internal/policy/      # image policy evaluation
internal/reconciler/  # patching YAML & GitOps commit
internal/git/         # GitHub App token + push/PR
internal/applier/     # optional compose runner
internal/config/      # config & policy loaders
internal/state/       # local cache
policies/             # declarative image policies
configs/              # main config.yaml

## 🔑 Config basics
	•	configs/config.yaml: main settings
	•	policies/*.yml: declarative rules per app
	•	Apps: each app links to a Compose file and a policy

Example policy:
```
apiVersion: v1
kind: ImagePolicy
metadata:
  name: guilliman
spec:
  selector:
    semver:
      range: ">=1.2.0 <2.0.0"
      allowPrerelease: false
  constraints:
    arch: ["amd64"]
  rollout:
    minAge: "10m"
```

## 🚀 How it works
	1.	Watcher detects a new image in GHCR.
	2.	Policy evaluator checks if it matches semver, arch, signature, etc.
	3.	If approved, Reconciler patches the GitOps repo (image: → digest), commits, and pushes with a GitHub App identity.
	4.	Loop or Applier on the homelab pulls the repo, decrypts secrets, and redeploys the stack.

⸻

## 🔒 Security
	•	Uses a GitHub App for ephemeral push tokens.
	•	Branch protection recommended on main.
	•	Secrets encrypted with SOPS + age, never stored in plain Git.
	•	Images pinned by digest in GitOps.
	•	Optional: cosign for signature verification.

⸻

## 🛠️ Roadmap
	•	MVP: GHCR polling → semver policy → patch Compose → push to main
	•	Add minAge & arch constraints
	•	Applier: SOPS decrypt + podman-compose
	•	PR mode instead of push
	•	Cosign signature verification
	•	Trivy vulnerability checks
	•	/healthz + /metrics

⸻

## 📜 License

MIT.
