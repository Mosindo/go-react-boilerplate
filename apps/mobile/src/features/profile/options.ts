import type { Gender } from "../../api/types";
import { strings } from "../../i18n/strings";
import type { ChipOption } from "../../shared/form";

export const GENDER_OPTIONS: ChipOption<Gender>[] = [
  { value: "woman", label: strings.gender.woman },
  { value: "man", label: strings.gender.man },
  { value: "nonbinary", label: strings.gender.nonbinary }
];

export function toggleInList<T>(list: T[], value: T, max = Infinity): T[] {
  if (list.includes(value)) {
    return list.filter((item) => item !== value);
  }
  return list.length >= max ? list : [...list, value];
}
