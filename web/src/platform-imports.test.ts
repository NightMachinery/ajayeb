// Source-level execution boundaries, independent of bundle chunk names.
import { existsSync, globSync, mkdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
import { expect, it } from 'vitest';

type Edge = { from: string; to: string; kind: 'eager' | 'lazy' };
const src = fileURLToPath(new URL('.', import.meta.url));
const golden = new URL('./testdata/platform-imports.json', import.meta.url);
const entries = ['routes/+layout.svelte', 'routes/+page.svelte', 'routes/play/[code]/+page.svelte'];

function imports(file: string, input?: string): { specifier: string; kind: Edge['kind'] }[] {
  let text = input ?? readFileSync(path.join(src, file), 'utf8');
  if (file.endsWith('.svelte')) {
    const component = parse(text, { modern: true });
    text = [component.instance, component.module]
      .flatMap((script) => {
        if (!script) return [];
        // Svelte supplies offsets at runtime, while its ESTree Program
        // declaration omits them. Check that boundary before slicing.
        const program = script.content;
        if (
          !('start' in program) ||
          !('end' in program) ||
          typeof program.start !== 'number' ||
          typeof program.end !== 'number'
        )
          throw new Error(`missing script offsets in ${file}`);
        return [text.slice(program.start, program.end)];
      })
      .join('\n');
  }
  const ast = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, ts.ScriptKind.TS);
  const found: ReturnType<typeof imports> = [];
  const add = (specifier: string, kind: Edge['kind']) => found.push({ specifier, kind });
  function visit(node: ts.Node): void {
    if (ts.isImportDeclaration(node) && ts.isStringLiteral(node.moduleSpecifier)) {
      // Binding-level type imports can emit `import {} from ...` under
      // verbatimModuleSyntax. Only a whole import-type declaration is erased.
      if (!node.importClause?.isTypeOnly) add(node.moduleSpecifier.text, 'eager');
    } else if (
      ts.isExportDeclaration(node) &&
      !node.isTypeOnly &&
      node.moduleSpecifier &&
      ts.isStringLiteral(node.moduleSpecifier)
    ) {
      add(node.moduleSpecifier.text, 'eager');
    } else if (ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword) {
      const arg = node.arguments[0];
      if (!arg || !ts.isStringLiteral(arg)) throw new Error(`nonliteral runtime import in ${file}`);
      add(arg.text, 'lazy');
    } else if (ts.isCallExpression(node) && node.expression.getText(ast) === 'import.meta.glob') {
      const arg = node.arguments[0];
      if (!arg || !ts.isStringLiteral(arg)) throw new Error(`unsupported import glob in ${file}`);
      const options = node.arguments[1];
      const eager =
        options &&
        ts.isObjectLiteralExpression(options) &&
        options.properties.some(
          (property) =>
            ts.isPropertyAssignment(property) &&
            property.name.getText(ast) === 'eager' &&
            property.initializer.kind === ts.SyntaxKind.TrueKeyword
        );
      for (const match of globSync(path.resolve(src, path.dirname(file), arg.text))) {
        add(
          './' + path.relative(path.resolve(src, path.dirname(file)), match),
          eager ? 'eager' : 'lazy'
        );
      }
    }
    ts.forEachChild(node, visit);
  }
  visit(ast);
  return found;
}

function resolve(from: string, specifier: string): string {
  const clean = specifier.split('?')[0]!;
  if (!clean.startsWith('.') && !clean.startsWith('$lib/')) return `external:${clean}`;
  const base = clean.startsWith('$lib/')
    ? path.join(src, 'lib', clean.slice(5))
    : path.resolve(src, path.dirname(from), clean);
  const target = [base, `${base}.ts`, `${base}.js`, `${base}/index.ts`, `${base}/index.js`].find(
    (candidate) => existsSync(candidate) && statSync(candidate).isFile()
  );
  if (!target) throw new Error(`unresolved runtime edge ${from} -> ${specifier}`);
  return path.relative(src, target).split(path.sep).join('/');
}

function graph(entry: string) {
  const pending = [entry];
  const seen = new Set<string>();
  const edges: Edge[] = [];
  while (pending.length) {
    const file = pending.pop()!;
    if (seen.has(file)) continue;
    seen.add(file);
    if (!/\.(svelte|ts|js)$/.test(file)) continue;
    for (const { specifier, kind } of imports(file)) {
      const to = resolve(file, specifier);
      edges.push({ from: file, to, kind });
      if (!to.startsWith('external:')) pending.push(to);
    }
  }
  const unique = [...new Map(edges.map((e) => [JSON.stringify(e), e])).values()];
  unique.sort((a, b) => JSON.stringify(a).localeCompare(JSON.stringify(b)));
  const eager = new Set([entry]);
  for (let size = -1; size !== eager.size;) {
    size = eager.size;
    for (const e of unique) if (e.kind === 'eager' && eager.has(e.from)) eager.add(e.to);
  }
  return { entry, eager: [...eager].sort(), edges: unique };
}

it('pins root, Home and Duel runtime import edges and excludes eager Haftgane code', () => {
  const actual = entries.map(graph);
  for (const entry of actual)
    expect(entry.eager.filter((file) => /haftgane/i.test(file))).toEqual([]);
  const bytes = JSON.stringify(actual, null, 2) + '\n';
  if (process.env.UPDATE_PLATFORM_BASELINES === '1') {
    mkdirSync(new URL('./testdata/', import.meta.url), { recursive: true });
    writeFileSync(golden, bytes);
  }
  if (bytes !== readFileSync(golden, 'utf8')) {
    throw new Error('Entry imports changed; review eager versus lazy edges before regenerating.');
  }
});

it('retains module side effects from binding-level type imports and empty re-exports', () => {
  expect(
    imports(
      'fixture.ts',
      `
    import type { A } from './erased';
    import { type B } from './import-side-effect';
    export type { C } from './also-erased';
    export { type D } from './export-side-effect';
    export {} from './empty-re-export';
    void import('./lazy');
  `
    )
  ).toEqual([
    { specifier: './import-side-effect', kind: 'eager' },
    { specifier: './export-side-effect', kind: 'eager' },
    { specifier: './empty-re-export', kind: 'eager' },
    { specifier: './lazy', kind: 'lazy' }
  ]);
  expect(() => imports('fixture.ts', 'void import(variable);')).toThrow(
    'nonliteral runtime import'
  );
});
