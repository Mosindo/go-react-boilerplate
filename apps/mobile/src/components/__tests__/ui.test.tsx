import React from "react";
import { fireEvent, render, screen } from "@testing-library/react-native";
import { Button, Chip, CountBadge, MessageItem, Notice, ThemeProvider } from "../../shared/ui";

const wrap = (node: React.ReactElement) => render(<ThemeProvider forceScheme="light">{node}</ThemeProvider>);

describe("Button", () => {
  it("calls onPress and exposes its label to assistive tech", () => {
    const onPress = jest.fn();
    wrap(<Button label="Continuer" onPress={onPress} />);
    fireEvent.press(screen.getByRole("button", { name: "Continuer" }));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("does not fire while loading or disabled", () => {
    const onPress = jest.fn();
    wrap(
      <>
        <Button label="Chargement" loading onPress={onPress} />
        <Button disabled label="Désactivé" onPress={onPress} />
      </>
    );
    fireEvent.press(screen.getByRole("button", { name: "Chargement" }));
    fireEvent.press(screen.getByRole("button", { name: "Désactivé" }));
    expect(onPress).not.toHaveBeenCalled();
  });
});

describe("Chip", () => {
  it("reports its checked state", () => {
    const onPress = jest.fn();
    wrap(<Chip label="Randonnée" onPress={onPress} selected />);
    const chip = screen.getByRole("checkbox", { name: "Randonnée" });
    expect(chip.props.accessibilityState).toMatchObject({ checked: true });
    fireEvent.press(chip);
    expect(onPress).toHaveBeenCalled();
  });
});

describe("CountBadge", () => {
  it("hides at zero and caps large counts", () => {
    wrap(<CountBadge count={0} />);
    expect(screen.queryByLabelText(/non lus/)).toBeNull();
    wrap(<CountBadge count={150} />);
    expect(screen.getByText("99+")).toBeTruthy();
  });
});

describe("MessageItem", () => {
  it("shows delivery status only on my messages", () => {
    wrap(<MessageItem body="Salut" mine status="read" time="10:00" />);
    expect(screen.getByText("10:00 · Lu")).toBeTruthy();
    wrap(<MessageItem body="Coucou" mine={false} time="10:01" />);
    expect(screen.getByText("10:01")).toBeTruthy();
  });
});

describe("Notice", () => {
  it("announces errors as alerts", () => {
    wrap(<Notice message="Oups" tone="danger" />);
    expect(screen.getByRole("alert")).toBeTruthy();
  });
});
