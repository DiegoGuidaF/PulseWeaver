/**
 * AlertDialog — a destructive-action confirmation dialog built on top of
 * the existing Dialog primitive (which uses @radix-ui/react-dialog).
 *
 * Exposes the same surface as the shadcn/ui AlertDialog so that consumers
 * don't notice the implementation detail. We do not depend on
 * @radix-ui/react-alert-dialog because it is not installed.
 */
import * as React from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

// Re-export root + trigger under the AlertDialog name so imports stay tidy.
const AlertDialog = Dialog;

// The trigger is just a slot — pass any element as the trigger child.
const AlertDialogTrigger = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement>
>(({ children, ...props }, ref) => (
  <button ref={ref} {...props}>
    {children}
  </button>
));
AlertDialogTrigger.displayName = "AlertDialogTrigger";

// Content wrapper — same as DialogContent but without the X close button.
const AlertDialogContent = React.forwardRef<
  React.ElementRef<typeof DialogContent>,
  React.ComponentPropsWithoutRef<typeof DialogContent>
>(({ className, ...props }, ref) => (
  <DialogContent
    ref={ref}
    showCloseButton={false}
    className={cn(className)}
    {...props}
  />
));
AlertDialogContent.displayName = "AlertDialogContent";

const AlertDialogHeader = DialogHeader;
const AlertDialogFooter = DialogFooter;
const AlertDialogTitle = DialogTitle;
const AlertDialogDescription = DialogDescription;

// Cancel button — closes the dialog via the Dialog close mechanism.
const AlertDialogCancel = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement>
>(({ className, children, onClick, ...props }, ref) => (
  <Button
    ref={ref}
    variant="outline"
    className={cn(className)}
    onClick={onClick}
    {...props}
  >
    {children ?? "Cancel"}
  </Button>
));
AlertDialogCancel.displayName = "AlertDialogCancel";

// Action button — styled destructive by default.
const AlertDialogAction = React.forwardRef<
  HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement>
>(({ className, children, ...props }, ref) => (
  <Button
    ref={ref}
    variant="destructive"
    className={cn(className)}
    {...props}
  >
    {children ?? "Continue"}
  </Button>
));
AlertDialogAction.displayName = "AlertDialogAction";

export {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogFooter,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogCancel,
  AlertDialogAction,
};
