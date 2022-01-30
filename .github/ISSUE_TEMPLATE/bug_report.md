---
name: Bug report
about: Template for bug reports.
title: ''
labels: ''
assignees: ''

---

#### Read the [FAQ](https://wiki.jfa-go.com/docs/faq/) first!

**Describe the bug**

Describe the problem, and what you would expect if it isn't clear already.

**To Reproduce**

What to do to reproduce the problem.

**Logs**

**If you're using a build with a tray icon, right-click on it and press "Open logs" to access your logs.**

When you notice the problem, check the output of `jfa-go` or get the logs by pressing the "Logs" button in the Settings tab. If the problem is not obvious (e.g a panic (red text) or 'ERROR' log), re-run jfa-go with the `-debug` argument and reproduce the problem. You should then take a screenshot of the output, or paste it here, preferably between \`\`\` tags (e.g \`\`\``Log here`\`\`\`). Remember to censor any personal information.


If nothing catches your eye in the log, access the admin page via your browser, go into the console (Right click > Inspect Element > Console), refresh, reproduce the problem then paste the output here in the same way as above.

**Configuration**

If you see it as necessary, include relevant sections of your `config.ini`, for example, include `[email]` and `[smtp]|[mailgun]` if you have an email issue.

**Platform/Version**

Include the platform jfa-go is running on (e.g Windows, Linux, Docker), the version (first line of output by `jfa-go` or Settings>About in web UI), and if necessary the browser version and platform.

