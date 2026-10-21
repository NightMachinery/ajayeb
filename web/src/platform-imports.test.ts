// Source-level execution boundaries, independent of bundle chunk names.
import { existsSync, globSync, mkdirSync, readFileSync, statSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse } from 'svelte/compiler';
import ts from 'typescript';
import { expect, it } from 'vitest';

type ImportKind = 'eager' | 'lazy';
type EagerBaseline = { entry: string; eager: string[] };
type EagerGraph = EagerBaseline & { parents: Map<string, string | null> };
const src = fileURLToPath(new URL('.', import.meta.url));
const golden = new URL('./testdata/platform-imports.json', import.meta.url);
const entries = ['routes/+layout.svelte', 'routes/+page.svelte', 'routes/play/[code]/+page.svelte'];

function imports(file: string, input?: string): { specifier: string; kind: ImportKind }[] {
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
  const add = (specifier: string, kind: ImportKind) => found.push({ specifier, kind });
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

// Keep predecessor paths in memory only. Breadth-first traversal makes
// each reported import chain shortest; sorted neighbours make ties stable.
function eagerGraph(entry: string, dependencies: (file: string) => string[]): EagerGraph {
  const pending = [entry];
  const parents = new Map<string, string | null>([[entry, null]]);
  for (const file of pending) {
    for (const to of dependencies(file).sort()) {
      if (parents.has(to)) continue;
      parents.set(to, file);
      pending.push(to);
    }
  }
  return { entry, eager: [...parents.keys()].sort(), parents };
}

function graph(entry: string): EagerGraph {
  return eagerGraph(entry, (file) => {
    if (file.startsWith('external:') || !/\.(svelte|ts|js)$/.test(file)) return [];
    return imports(file)
      .filter(({ kind }) => kind === 'eager')
      .map(({ specifier }) => resolve(file, specifier));
  });
}

function importPath(graph: EagerGraph, module: string): string {
  const chain = [module];
  for (let from = graph.parents.get(module); from != null; from = graph.parents.get(from))
    chain.push(from);
  return chain.reverse().join(' -> ');
}

function checkEagerBaseline(expected: EagerBaseline, actual: EagerGraph): void {
  if (expected.entry !== actual.entry)
    throw new Error(`entry baseline mismatch: ${expected.entry} versus ${actual.entry}`);
  const wanted = new Set(expected.eager);
  const unexpected = actual.eager.find((module) => !wanted.has(module) || /haftgane/i.test(module));
  if (unexpected)
    throw new Error(
      `Unexpected eager module ${unexpected} for ${actual.entry}; import path: ${importPath(actual, unexpected)}`
    );
  const present = new Set(actual.eager);
  const missing = expected.eager.find((module) => !present.has(module));
  if (missing)
    throw new Error(
      `Missing eager module ${missing} for ${actual.entry}; review before regenerating its sorted module set.`
    );
  if (JSON.stringify(expected.eager) !== JSON.stringify(actual.eager))
    throw new Error(`Eager module set for ${actual.entry} must be sorted and unique.`);
}

it('pins root, Home and Duel eager modules and reports unexpected import paths', () => {
  const actual = entries.map(graph);
  if (process.env.UPDATE_PLATFORM_BASELINES === '1') {
    mkdirSync(new URL('./testdata/', import.meta.url), { recursive: true });
    writeFileSync(
      golden,
      JSON.stringify(
        actual.map(({ entry, eager }) => ({ entry, eager })),
        null,
        2
      ) + '\n'
    );
  }
  const expected = JSON.parse(readFileSync(golden, 'utf8')) as EagerBaseline[];
  expect(expected.length).toBe(actual.length);
  for (const [index, entry] of actual.entries()) checkEagerBaseline(expected[index]!, entry);
});

it('names the shortest eager import path and permits lazy-only module changes', () => {
  const dependencies: Record<string, string[]> = {
    root: ['bridge', 'detour'],
    bridge: ['new-module'],
    detour: ['longer'],
    longer: ['new-module']
  };
  const actual = eagerGraph('root', (file) => dependencies[file] ?? []);
  expect(() =>
    checkEagerBaseline({ entry: 'root', eager: ['bridge', 'detour', 'longer', 'root'] }, actual)
  ).toThrow(
    'Unexpected eager module new-module for root; import path: root -> bridge -> new-module'
  );
  expect(() => checkEagerBaseline({ entry: 'root', eager: actual.eager }, actual)).not.toThrow();
  expect(
    imports('fixture.ts', "void import('./lazy-only');").filter(({ kind }) => kind === 'eager')
  ).toEqual([]);
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
