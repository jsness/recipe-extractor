import { IngredientGroup } from "../types";

const EPSILON = 0.00001;
const FRACTION_DENOMINATORS = [2, 3, 4, 5, 6, 8, 10, 12, 16];
const AMOUNT_BOUNDARY = "(?=\\s|$|-|–|—)";
const VULGAR_FRACTIONS: Record<string, number> = {
  "¼": 1 / 4,
  "½": 1 / 2,
  "¾": 3 / 4,
  "⅐": 1 / 7,
  "⅑": 1 / 9,
  "⅒": 1 / 10,
  "⅓": 1 / 3,
  "⅔": 2 / 3,
  "⅕": 1 / 5,
  "⅖": 2 / 5,
  "⅗": 3 / 5,
  "⅘": 4 / 5,
  "⅙": 1 / 6,
  "⅚": 5 / 6,
  "⅛": 1 / 8,
  "⅜": 3 / 8,
  "⅝": 5 / 8,
  "⅞": 7 / 8,
};

type ParsedAmount = {
  value: number;
  startIndex: number;
  endIndex: number;
};

type ParsedAmountRange = {
  min: ParsedAmount;
  max: ParsedAmount;
  separator: string;
  startIndex: number;
  endIndex: number;
};

type ParsedQuantity =
  | { kind: "single"; amount: ParsedAmount }
  | { kind: "range"; range: ParsedAmountRange };

const roundTo = (value: number, places: number) => {
  const scale = 10 ** places;
  return Math.round(value * scale) / scale;
};

const parseNumericValue = (value: string) => Number.parseFloat(value);

const parseAmountAt = (value: string, startIndex: number): ParsedAmount | null => {
  const text = value.slice(startIndex);

  const mixedNumber = text.match(new RegExp(`^(\\d+)\\s+(\\d+)\\/(\\d+)${AMOUNT_BOUNDARY}`));
  if (mixedNumber) {
    const whole = parseNumericValue(mixedNumber[1]);
    const numerator = parseNumericValue(mixedNumber[2]);
    const denominator = parseNumericValue(mixedNumber[3]);
    if (denominator > 0) {
      return {
        value: whole + numerator / denominator,
        startIndex,
        endIndex: startIndex + mixedNumber[0].length,
      };
    }
  }

  const mixedVulgarFraction = text.match(new RegExp(`^(\\d+)\\s*([¼½¾⅐⅑⅒⅓⅔⅕⅖⅗⅘⅙⅚⅛⅜⅝⅞])${AMOUNT_BOUNDARY}`));
  if (mixedVulgarFraction) {
    return {
      value: parseNumericValue(mixedVulgarFraction[1]) + VULGAR_FRACTIONS[mixedVulgarFraction[2]],
      startIndex,
      endIndex: startIndex + mixedVulgarFraction[0].length,
    };
  }

  const fraction = text.match(new RegExp(`^(\\d+)\\/(\\d+)${AMOUNT_BOUNDARY}`));
  if (fraction) {
    const numerator = parseNumericValue(fraction[1]);
    const denominator = parseNumericValue(fraction[2]);
    if (denominator > 0) {
      return {
        value: numerator / denominator,
        startIndex,
        endIndex: startIndex + fraction[0].length,
      };
    }
  }

  const vulgarFraction = text.match(new RegExp(`^([¼½¾⅐⅑⅒⅓⅔⅕⅖⅗⅘⅙⅚⅛⅜⅝⅞])${AMOUNT_BOUNDARY}`));
  if (vulgarFraction) {
    return {
      value: VULGAR_FRACTIONS[vulgarFraction[1]],
      startIndex,
      endIndex: startIndex + vulgarFraction[0].length,
    };
  }

  const decimal = text.match(new RegExp(`^(?:\\d+(?:\\.\\d+)?|\\.\\d+)${AMOUNT_BOUNDARY}`));
  if (decimal) {
    return {
      value: parseNumericValue(decimal[0]),
      startIndex,
      endIndex: startIndex + decimal[0].length,
    };
  }

  return null;
};

const parseLeadingQuantity = (value: string): ParsedQuantity | null => {
  const leadingSpaceLength = value.length - value.trimStart().length;
  const amount = parseAmountAt(value, leadingSpaceLength);
  if (!amount) {
    return null;
  }

  const rangeSeparator = value.slice(amount.endIndex).match(/^(\s*(?:-|–|—|to)\s*)/i);
  if (!rangeSeparator) {
    return { kind: "single", amount };
  }

  const max = parseAmountAt(value, amount.endIndex + rangeSeparator[0].length);
  if (!max) {
    return { kind: "single", amount };
  }

  return {
    kind: "range",
    range: {
      min: amount,
      max,
      separator: rangeSeparator[1],
      startIndex: amount.startIndex,
      endIndex: max.endIndex,
    },
  };
};

const formatScaledAmount = (value: number) => {
  const roundedInteger = Math.round(value);
  if (Math.abs(value - roundedInteger) < EPSILON) {
    return String(roundedInteger);
  }

  const whole = Math.floor(value);
  const remainder = value - whole;

  for (const denominator of FRACTION_DENOMINATORS) {
    const numerator = Math.round(remainder * denominator);
    if (numerator > 0 && Math.abs(remainder - numerator / denominator) < EPSILON) {
      if (numerator === denominator) {
        return String(whole + 1);
      }
      return whole > 0 ? `${whole} ${numerator}/${denominator}` : `${numerator}/${denominator}`;
    }
  }

  return String(roundTo(value, 2)).replace(/\.0+$/, "");
};

const scaleIngredientItem = (item: string, scale: number) => {
  const quantity = parseLeadingQuantity(item);
  if (!quantity) {
    return item;
  }

  if (quantity.kind === "single") {
    const { amount } = quantity;
    return `${item.slice(0, amount.startIndex)}${formatScaledAmount(amount.value * scale)}${item.slice(amount.endIndex)}`;
  }

  const { range } = quantity;
  return [
    item.slice(0, range.startIndex),
    formatScaledAmount(range.min.value * scale),
    range.separator,
    formatScaledAmount(range.max.value * scale),
    item.slice(range.endIndex),
  ].join("");
};

export const scaleIngredientGroups = (
  ingredientGroups: IngredientGroup[],
  scale: number,
) => {
  if (Math.abs(scale - 1) < EPSILON) {
    return ingredientGroups;
  }

  return ingredientGroups.map((group) => ({
    ...group,
    items: group.items.map((item) => scaleIngredientItem(item, scale)),
  }));
};

export const extractServingCount = (recipeYield?: string) => {
  if (!recipeYield) {
    return null;
  }

  const match = recipeYield.match(/(?:serves|servings?|yield)?\s*(\d+(?:\.\d+)?)/i);
  if (!match) {
    return null;
  }

  const servings = parseNumericValue(match[1]);
  return Number.isFinite(servings) && servings > 0 ? servings : null;
};
