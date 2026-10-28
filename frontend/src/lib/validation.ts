import { z } from 'zod'

/** PostgreSQL INTEGER values must stay in range even when JavaScript could hold more. */
export const MAX_INTEGER = 2_147_483_647

/** Parse whole decimal input without silently rounding, accepting exponents, or coercing blanks. */
function integerInput(minimum: number) {
  return z
    .string()
    .trim()
    .regex(/^\d+$/, 'Enter a whole number.')
    .transform(Number)
    .pipe(
      z
        .number()
        .int()
        .min(minimum, minimum ? 'Quantity must be greater than zero.' : 'Price cannot be negative.')
        .max(MAX_INTEGER, 'Value is too large.'),
    )
}

/** Shared positive-integer rule is used by stock and order quantity fields. */
export const quantitySchema = integerInput(1)

/** Catalog edits omit stock and keep the SKU stable after creation. */
export const productSchema = z.object({
  sku: z.string().trim().min(1, 'SKU is required.').max(64),
  name: z.string().trim().min(1, 'Name is required.').max(160),
  price: integerInput(0),
  is_active: z.boolean(),
})

/** Stock directions and notes are validated before enabling the network mutation. */
export const stockSchema = z.object({
  type: z.enum(['in', 'out']),
  qty: quantitySchema,
  note: z.string().trim().max(500, 'Use at most 500 characters.'),
})

/** Duplicate SKUs receive errors on their specific item row, rather than a generic form failure. */
export const orderSchema = z
  .object({
    customer: z.string().trim().min(1, 'Customer name is required.').max(160),
    items: z
      .array(
        z.object({
          sku: z.string().trim().min(1, 'Choose a product.').max(64),
          qty: quantitySchema,
        }),
      )
      .min(1, 'Add at least one item.')
      .max(100),
  })
  .superRefine((input, ctx) => {
    const seen = new Set<string>()
    input.items.forEach((item, index) => {
      if (seen.has(item.sku))
        ctx.addIssue({
          code: 'custom',
          path: ['items', index, 'sku'],
          message: 'This SKU is already in the order.',
        })
      seen.add(item.sku)
    })
  })

/** Convert schema paths into stable field keys usable by small controlled forms. */
export function fieldErrors(error: z.ZodError): Record<string, string> {
  return Object.fromEntries(error.issues.map((issue) => [issue.path.join('.'), issue.message]))
}
