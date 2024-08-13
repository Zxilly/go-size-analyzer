import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { FileSelector } from "./FileSelector.tsx";

function createFileWithSize(sizeInBytes: number, fileName = "test.txt", fileType = "text/plain") {
  const buffer = new ArrayBuffer(sizeInBytes);
  const blob = new Blob([buffer], { type: fileType });
  return new File([blob], fileName, { type: fileType });
}

describe("fileSelector", () => {
  it("should render correctly", () => {
    const mockHandler = vi.fn();
    const { getByText } = render(<FileSelector handler={mockHandler} />);
    expect(getByText("Click or drag file to analyze")).toBeInTheDocument();
  });

  it("should call handler when file size is within limit", async () => {
    const mockHandler = vi.fn();
    const { getByTestId } = render(<FileSelector handler={mockHandler} />);
    const file = createFileWithSize(1024 * 1024 * 29);
    await userEvent.upload(getByTestId("file-selector"), file);
    expect(mockHandler).toHaveBeenCalledWith(file);
  });

  it("should not call handler when file size exceeds limit", async () => {
    const mockHandler = vi.fn();
    const { getByTestId } = render(<FileSelector handler={mockHandler} />);
    const file = createFileWithSize(1024 * 1024 * 31);

    await userEvent.upload(getByTestId("file-selector"), file);
    expect(mockHandler).not.toHaveBeenCalled();
  });

  it("should call handler when file size exceeds limit and user chooses to continue", async () => {
    const mockHandler = vi.fn();
    const { getByTestId, getByText } = render(<FileSelector handler={mockHandler} />);
    const file = createFileWithSize(1024 * 1024 * 31);

    await userEvent.upload(getByTestId("file-selector"), file);
    await userEvent.click(getByText("Continue"));

    expect(mockHandler).toHaveBeenCalledWith(file);
  });

  it("should not call handler when file size exceeds limit and user chooses to cancel", async () => {
    const mockHandler = vi.fn();
    const { getByTestId, getByText } = render(<FileSelector handler={mockHandler} />);
    const file = createFileWithSize(1024 * 1024 * 31);

    await userEvent.upload(getByTestId("file-selector"), file as File);
    await userEvent.click(getByText("Cancel"));

    expect(mockHandler).not.toHaveBeenCalled();
  });
});
