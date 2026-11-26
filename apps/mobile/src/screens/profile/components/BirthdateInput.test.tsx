import React, { useState } from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { BirthdateInput, partsFromIso, type BirthdateParts } from "./BirthdateInput";

function Harness({ onValue }: { onValue: (v: BirthdateParts) => void }) {
  const [value, setValue] = useState<BirthdateParts>({ day: "", month: "", year: "" });
  return (
    <BirthdateInput
      onChange={(v) => {
        setValue(v);
        onValue(v);
      }}
      value={value}
    />
  );
}

describe("BirthdateInput", () => {
  it("keeps digits only", () => {
    const onValue = jest.fn();
    render(<Harness onValue={onValue} />);
    fireEvent.changeText(screen.getByTestId("birthdate-day"), "1a5");
    expect(onValue).toHaveBeenLastCalledWith({ day: "15", month: "", year: "" });
  });

  it("parses ISO dates into parts", () => {
    expect(partsFromIso("1994-03-07")).toEqual({ day: "07", month: "03", year: "1994" });
    expect(partsFromIso("")).toEqual({ day: "", month: "", year: "" });
  });
});
