// Shapes shared by the message tables and t().

/** The forms of a count-dependent message. 'other' is required. */
export type PluralForms = { readonly other: string } & {
  readonly [form in Intl.LDMLPluralRule | `=${number}`]?: string;
};

export type Message = string | PluralForms;

/** A message table: one area's messages for one locale. */
export type Messages = Readonly<Record<string, Message>>;
