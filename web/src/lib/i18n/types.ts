// Shapes shared by the message tables and t().

/** The forms of a count-dependent message. 'other' is required. */
export type PluralForms = { readonly other: string } & {
  readonly [form in Intl.LDMLPluralRule | `=${number}`]?: string;
};

export type Message = string | PluralForms;

/** A message table: one area's messages for one locale. */
export type Messages = Readonly<Record<string, Message>>;

/** Another locale's version of one English area table: any subset of its
 *  keys (t() falls back to English per key), each with its own wording. An
 *  extra or misspelt key is a type error through `satisfies`. */
export type Translation<T extends Messages> = {
  readonly [K in keyof T]?: T[K] extends string ? string : PluralForms;
};
