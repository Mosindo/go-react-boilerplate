import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react-native";
import { FeedbackProvider, useFeedback } from "../components/Feedback";
import { ProfileBody } from "../components/profile";
import { Button, EmptyState } from "../components/ui";
import type { Card } from "../api/types";

jest.mock("@expo/vector-icons", () => ({ Ionicons: () => null }));
jest.mock("../components/AuthImage", () => ({ AuthImage: () => null, Avatar: () => null, clearImageCache: () => undefined }));

const card: Card = {
  userId: "u1",
  firstName: "Camille",
  age: 29,
  gender: "woman",
  bio: "Passionnée de cuisine",
  city: "Paris",
  distanceKm: 10,
  interests: [{ id: 1, slug: "cuisine", label: "Cuisine" }],
  photos: []
};

describe("ui", () => {
  it("Button fires onPress, and not while loading", () => {
    const onPress = jest.fn();
    const { rerender } = render(<Button label="Continuer" onPress={onPress} />);
    fireEvent.press(screen.getByLabelText("Continuer"));
    expect(onPress).toHaveBeenCalledTimes(1);
    rerender(<Button label="Continuer" onPress={onPress} loading />);
    fireEvent.press(screen.getByLabelText("Continuer"));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it("EmptyState renders its action", () => {
    const onAction = jest.fn();
    render(<EmptyState icon="heart" title="Vide" message="Rien ici" actionLabel="Actualiser" onAction={onAction} />);
    fireEvent.press(screen.getByText("Actualiser"));
    expect(onAction).toHaveBeenCalled();
  });

  it("ProfileBody shows only public, approximate information", () => {
    render(<ProfileBody card={card} />);
    expect(screen.getByText("Camille, 29")).toBeTruthy();
    expect(screen.getByText(/à environ 10 km/)).toBeTruthy();
    expect(screen.getByText("Passionnée de cuisine")).toBeTruthy();
    expect(screen.getByText("Cuisine")).toBeTruthy();
  });

  it("FeedbackProvider confirm resolves with the user's choice", async () => {
    let answer: boolean | undefined;
    function Trigger() {
      const { confirm } = useFeedback();
      return <Button label="ask" onPress={() => void confirm({ title: "Sûr ?" }).then((v) => (answer = v))} />;
    }
    render(
      <FeedbackProvider>
        <Trigger />
      </FeedbackProvider>
    );
    fireEvent.press(screen.getByLabelText("ask"));
    fireEvent.press(await screen.findByTestId("confirm-yes"));
    await waitFor(() => expect(answer).toBe(true));
  });
});
