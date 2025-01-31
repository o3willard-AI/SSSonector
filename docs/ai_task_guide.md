# AI Task Guide for SSSonector

This guide explains how to effectively use the project context document with AI engineers for future development tasks.

## Starting a New Task Session

1. Begin each new task session by providing the AI with:
   - The contents of project_context.md
   - The specific task requirements
   - Any additional relevant context

Example:
```markdown
I'd like to work on [task description]. Here's the project context:

[paste contents of project_context.md]

Specific task requirements:
1. [requirement 1]
2. [requirement 2]
...
```

## Task Format

Structure your tasks in this format for optimal AI understanding:

```markdown
Task: [Clear, concise description]

Relevant Sections:
- [Reference specific sections from project_context.md]
- [Any additional context specific to this task]

Requirements:
1. [Specific requirement]
2. [Specific requirement]
...

Expected Outcome:
- [Clear description of what success looks like]
- [Any specific deliverables]

Additional Context:
- [Any relevant code snippets]
- [Related documentation]
- [Previous work references]
```

## Best Practices

1. **Context Loading**
   - Always provide project_context.md at the start
   - Reference specific sections relevant to the task
   - Include any additional context not covered in the main document

2. **Task Scoping**
   - Break large tasks into smaller, manageable chunks
   - Clearly define the boundaries of each task
   - Specify any dependencies or prerequisites

3. **Documentation Updates**
   - If the task results in significant changes, provide instructions to update project_context.md
   - Keep the context document current with new features or architectural changes

4. **Code References**
   - When referencing existing code, specify the file path and relevant sections
   - Include any related configuration changes needed

## Example Task Sessions

### Feature Addition
```markdown
Task: Add bandwidth throttling configuration to client mode

Context: [project_context.md content]

Relevant Sections:
- Configuration Management
- Performance Improvements
- Client Mode Configuration

Requirements:
1. Add bandwidth throttling settings to client config
2. Implement throttling logic
3. Update documentation

Expected Outcome:
- New configuration options in client config
- Working bandwidth throttling
- Updated documentation
```

### Bug Fix
```markdown
Task: Fix connection persistence issue in client mode

Context: [project_context.md content]

Relevant Sections:
- Tunnel Management
- Client Mode Configuration
- Testing

Requirements:
1. Investigate connection drops
2. Implement fix
3. Add test cases

Expected Outcome:
- Stable connection maintenance
- Automated tests for the fix
- Updated documentation if needed
```

## Maintaining Context

1. **Regular Updates**
   - After significant changes, update project_context.md
   - Keep the directory structure current
   - Update configuration examples

2. **Version Tracking**
   - Note any API changes
   - Document breaking changes
   - Keep installation instructions current

3. **Documentation Sync**
   - Ensure all documentation reflects current state
   - Update examples with new features
   - Keep testing procedures current

## Tips for Effective AI Interaction

1. **Clear Communication**
   - Be specific about requirements
   - Provide concrete examples
   - Reference existing patterns

2. **Iterative Development**
   - Break complex tasks into steps
   - Verify each step before proceeding
   - Keep track of progress

3. **Quality Assurance**
   - Include testing requirements
   - Specify validation criteria
   - Define acceptance criteria

## Troubleshooting

If the AI seems to miss context or make incorrect assumptions:

1. Check if relevant sections were properly highlighted
2. Provide more specific references to existing code
3. Break down the task into smaller, clearer steps
4. Include more specific examples or use cases

## Conclusion

This guide helps maintain consistent, efficient interaction with AI engineers while working on the SSSonector project. By following these guidelines, you can ensure that the AI has the necessary context to provide accurate and helpful assistance while maintaining the project's quality and consistency.
