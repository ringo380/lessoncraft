# LessonCraft Requirements Specification

## Overview

**LessonCraft** is an e-learning platform forked from `play-with-docker`, reimagined to deliver dynamic, browser-based technical education. It integrates interactive lessons, Docker-based live environments, and performance assessments, enabling educators to create engaging, hands-on learning experiences.

## 1. Functional Requirements

### 1.1 Lesson Management
- [ ] Support for Markdown-based lesson creation.
- [ ] Define contextual commands and variables inside lesson markdown files (e.g. `{{variable}}`, `%%command%%`, etc.).
- [ ] Versioning support for lessons.
- [ ] Ability to preview lessons with live rendering before publishing.
- [ ] Support linking lessons in a sequence to form a curriculum.

### 1.2 User Interface (UI/UX)
- [ ] Modern, responsive UI replacing original PWD UI with LessonCraft branding.
- [ ] Dark mode and accessibility features (screen reader compatibility, keyboard navigation).
- [ ] Lesson player view with split panel (instructions + terminal).
- [ ] Admin panel for educators to manage content and monitor learners.

### 1.3 Sandbox Environments
- [ ] Launch disposable Docker environments per lesson step.
- [ ] Automatically configure containers based on lesson metadata.
- [ ] Support multiple OS variants (e.g., Alpine, CentOS 7, Rocky Linux, Ubuntu).
- [ ] Volume and file mounting support for persistent exercises.
- [ ] Terminal session logging for assessment purposes.

### 1.4 Grading and Assessment
- [ ] Integrate command validation for exercises (e.g., regex or exact match validation).
- [ ] Track command history and output.
- [ ] Rubric-based or pass/fail assessment engine.
- [ ] Final exam with multiple choice, code input, and task-based grading.
- [ ] Export grades to CSV/JSON or API integration with LMS platforms.

### 1.5 User Management
- [ ] Basic authentication (email/password).
- [ ] Role-based access (Admin, Educator, Learner).
- [ ] Support for OAuth (Google, GitHub).
- [ ] User progress tracking with resume capability.

### 1.6 Analytics and Reporting
- [ ] Dashboard with lesson completions, average time spent, assessment scores.
- [ ] Exportable user activity logs.
- [ ] Completion certificates for learners.

## 2. Technical Requirements

### 2.1 Backend
- [ ] Replace or refactor any `github.com/play-with-docker/pwd` imports with `github.com/ringo380/lessoncraft/pwd`.
- [ ] Use Go modules with proper replacement paths for internal packages.
- [ ] RESTful API for lesson content, user data, and sandbox control.
- [ ] Redis for session and state storage.
- [ ] PostgreSQL for persistent user data and progress tracking.
- [ ] Docker SDK or API for managing containers.

### 2.2 Frontend
- [ ] React or Vue-based frontend.
- [ ] WebSocket support for real-time terminal and progress feedback.
- [ ] Markdown rendering with syntax highlighting and inline code execution where applicable.

### 2.3 DevOps and Infrastructure
- [ ] Docker Compose setup for local development and CI.
- [ ] GitHub Actions or equivalent CI/CD pipeline.
- [ ] Kubernetes-compatible deployment setup for production scalability.
- [ ] SSL and domain configuration for secure access.

## 3. Migration and Refactor Tasks
- [ ] Remove hardcoded references to `play-with-docker` and replace with `lessoncraft`.
- [ ] Audit and update all internal module references in `go.mod` and source files.
- [ ] Rewrite legacy services (e.g., PWD scheduler) to better support lesson-based container logic.
- [ ] Introduce a modular architecture (e.g., `lesson-engine`, `user-service`, `env-runner`).

## 4. Stretch Goals
- [ ] AI assistant integration for context-aware hints.
- [ ] Lesson marketplace for community-contributed content.
- [ ] GitHub Classroom / LMS integration (e.g., Canvas, Moodle).
- [ ] Multi-container scenarios (e.g., Kubernetes labs, microservices exercises).
- [ ] Offline/local mode for educators.

## 5. Non-Functional Requirements
- [ ] High availability for live environments.
- [ ] Fast container provisioning (< 5 seconds).
- [ ] Secure container isolation.
- [ ] Support minimum browser requirements (Chrome/Firefox latest two versions).
- [ ] GDPR and privacy compliance for user data.

---

*This document is a living specification. As LessonCraft evolves, the requirements herein should be reviewed and updated regularly.*